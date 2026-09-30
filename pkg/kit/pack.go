package kit

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"aoe2kit/pkg/aoe2"
)

type PackReport struct {
	Root            string                 `json:"root"`
	Output          string                 `json:"output"`
	Profile         string                 `json:"profile,omitempty"`
	SHASidecar      string                 `json:"sha256_sidecar,omitempty"`
	ManifestUpdated bool                   `json:"manifest_updated,omitempty"`
	FileCount       int                    `json:"file_count"`
	Bytes           int64                  `json:"bytes"`
	SHA256          string                 `json:"sha256"`
	Verified        bool                   `json:"verified"`
	Verification    aoe2.VerificationClaim `json:"verification"`
	VerifyError     string                 `json:"verify_error,omitempty"`
}

// PackProfile names an audience, because that is what actually decides what
// belongs in an archive.
//
//	full     everything tracked, plus the prebuilt binary — our own archive
//	handoff  source, data, and root-level documents for someone who has Go
//	sandbox  handoff plus the binary, for a container that can run but not compile
//	public   default-deny: exactly the files listed in release/public-files.json
type PackProfile string

const (
	ProfileFull    PackProfile = "full"
	ProfileHandoff PackProfile = "handoff"
	ProfileSandbox PackProfile = "sandbox"
	ProfilePublic  PackProfile = "public"
)

type PackOptions struct {
	Profile        PackProfile
	Exclude        []string // additional glob patterns, matched against the relative path
	SHASidecar     bool     // write <output>.sha256 next to the archive
	KeepBinary     *bool    // override the profile's binary decision
	UpdateManifest bool     // refresh KIT_MANIFEST.json before verifying
}

// excludes applies to the non-public profiles. The public profile has no
// exclusion rules: it ships only what release/public-files.json lists.
func (p PackProfile) excludes() []string {
	switch p {
	case ProfileHandoff, ProfileSandbox:
		return []string{"docs/**", "DEX_TASK_*.md", "AUDITS.md", "BRIEFING.md", "KIT_PARITY_ROADMAP.md"}
	default:
		return nil
	}
}

func (p PackProfile) includesBinary() bool {
	return p == ProfileFull || p == ProfileSandbox
}

// Pack keeps the original signature and behavior: a full archive.
func Pack(root, outputPath string) (PackReport, error) {
	return PackWithOptions(root, outputPath, PackOptions{Profile: ProfileFull})
}

func PackWithOptions(root, outputPath string, opts PackOptions) (PackReport, error) {
	if opts.Profile == "" {
		opts.Profile = ProfileFull
	}
	// The manifest is derived, so a stale snapshot is a chore rather than a
	// finding. Refusing to pack over it strands anyone who edits the kit they
	// were given, which is exactly what we want them to do.
	manifestUpdated := false
	if opts.UpdateManifest {
		generated, err := ManifestJSON()
		if err != nil {
			return PackReport{}, err
		}
		if err := os.WriteFile(filepath.Join(root, "KIT_MANIFEST.json"), append(generated, '\n'), 0o644); err != nil {
			return PackReport{}, err
		}
		manifestUpdated = true
	}
	verifyReport := Verify(root, false)
	if !verifyReport.OK() {
		return PackReport{
			Root:         root,
			Output:       outputPath,
			Verified:     false,
			Verification: aoe2.StructureVerification(false),
			VerifyError:  strings.Join(verifyReport.Errors, "; "),
		}, fmt.Errorf("refusing to pack invalid kit: %s", strings.Join(verifyReport.Errors, "; "))
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return PackReport{}, err
	}
	outputAbs, err := filepath.Abs(outputPath)
	if err != nil {
		return PackReport{}, err
	}
	keepBinary := opts.Profile.includesBinary()
	if opts.KeepBinary != nil {
		keepBinary = *opts.KeepBinary
	}
	excludes := append(append([]string{}, opts.Profile.excludes()...), opts.Exclude...)
	files, err := packFileListForProfile(rootAbs, outputAbs, opts.Profile, excludes, keepBinary)
	if err != nil {
		return PackReport{}, err
	}
	if err := os.MkdirAll(filepath.Dir(outputAbs), 0o755); err != nil {
		return PackReport{}, err
	}
	out, err := os.Create(outputAbs)
	if err != nil {
		return PackReport{}, err
	}
	zw := zip.NewWriter(out)
	for _, file := range files {
		if err := addZipFile(zw, rootAbs, file); err != nil {
			zw.Close()
			out.Close()
			return PackReport{}, err
		}
	}
	// Stamp what this archive is, so it can be verified on its own terms and
	// repacked by whoever receives it. Written straight into the archive rather
	// than onto the source tree.
	marker := nextMarker(rootAbs, opts.Profile, time.Now())
	markerJSON, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		zw.Close()
		out.Close()
		return PackReport{}, err
	}
	if err := addZipBytes(zw, ProfileMarkerName, append(markerJSON, '\n')); err != nil {
		zw.Close()
		out.Close()
		return PackReport{}, err
	}
	if err := zw.Close(); err != nil {
		out.Close()
		return PackReport{}, err
	}
	if err := out.Close(); err != nil {
		return PackReport{}, err
	}
	st, err := os.Stat(outputAbs)
	if err != nil {
		return PackReport{}, err
	}
	hash, err := fileSHA256(outputAbs)
	if err != nil {
		return PackReport{}, err
	}
	sidecar := ""
	if opts.SHASidecar {
		sidecar = outputAbs + ".sha256"
		line := fmt.Sprintf("%s  %s\n", hash, filepath.Base(outputAbs))
		if err := os.WriteFile(sidecar, []byte(line), 0o644); err != nil {
			return PackReport{}, err
		}
	}
	return PackReport{
		Root:            root,
		Output:          outputPath,
		Profile:         string(opts.Profile),
		SHASidecar:      sidecar,
		ManifestUpdated: manifestUpdated,
		FileCount:       len(files),
		Bytes:           st.Size(),
		SHA256:          hash,
		Verified:        true,
		Verification:    aoe2.StructureVerification(true),
		VerifyError:     "",
	}, nil
}

// excluded reports whether a relative path matches any exclusion glob. A
// pattern ending in /** also excludes the directory itself.
func excluded(rel string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.HasSuffix(pattern, "/**") {
			dir := strings.TrimSuffix(pattern, "/**")
			if rel == dir || strings.HasPrefix(rel, dir+"/") {
				return true
			}
			continue
		}
		if ok, _ := filepath.Match(pattern, rel); ok {
			return true
		}
		if ok, _ := filepath.Match(pattern, filepath.Base(rel)); ok {
			return true
		}
	}
	return false
}

func packFileList(rootAbs, outputAbs string, excludes []string, keepBinary bool) ([]string, error) {
	return packFileListWithAllowlist(rootAbs, outputAbs, excludes, keepBinary, nil)
}

func packFileListForProfile(rootAbs, outputAbs string, profile PackProfile, excludes []string, keepBinary bool) ([]string, error) {
	if profile != ProfilePublic {
		return packFileList(rootAbs, outputAbs, excludes, keepBinary)
	}
	allowlist, err := readPublicAllowlist(rootAbs)
	if err != nil {
		return nil, err
	}
	return packFileListWithAllowlist(rootAbs, outputAbs, excludes, keepBinary, allowlist)
}

func packFileListWithAllowlist(rootAbs, outputAbs string, excludes []string, keepBinary bool, allowlist map[string]bool) ([]string, error) {
	var files []string
	ignore, err := loadRootGitignore(rootAbs)
	if err != nil {
		return nil, err
	}
	walkErr := filepath.WalkDir(rootAbs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := d.Name()
		pathAbs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(rootAbs, pathAbs)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if allowlist != nil && rel != "." && !d.IsDir() && !allowlist[rel] {
			return nil
		}
		if rel != "." && excluded(rel, excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == "kit" && !keepBinary {
			return nil
		}
		// The marker is always written fresh; packing the inherited copy too
		// would put two entries of the same name in the archive.
		if rel == ProfileMarkerName {
			return nil
		}
		if rel != "." && !mustPackIgnoredPath(rel) && ignore.ignored(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if shouldSkipPackDir(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldSkipPackFile(name) {
			return nil
		}
		if pathAbs == outputAbs {
			return nil
		}
		files = append(files, pathAbs)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	sort.Strings(files)
	return files, nil
}

func mustPackIgnoredPath(rel string) bool {
	return rel == "kit"
}

type rootGitignore struct {
	patterns []gitignorePattern
}

type gitignorePattern struct {
	raw      string
	pattern  string
	anchored bool
	dirOnly  bool
}

func loadRootGitignore(rootAbs string) (rootGitignore, error) {
	data, err := os.ReadFile(filepath.Join(rootAbs, ".gitignore"))
	if err != nil {
		if os.IsNotExist(err) {
			return rootGitignore{}, nil
		}
		return rootGitignore{}, err
	}
	var out rootGitignore
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		line = filepath.ToSlash(line)
		p := gitignorePattern{raw: line}
		if strings.HasPrefix(line, "/") {
			p.anchored = true
			line = strings.TrimPrefix(line, "/")
		}
		if strings.HasSuffix(line, "/") {
			p.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}
		if line == "" {
			continue
		}
		p.pattern = line
		out.patterns = append(out.patterns, p)
	}
	return out, nil
}

func (g rootGitignore) ignored(rel string, isDir bool) bool {
	rel = filepath.ToSlash(strings.TrimPrefix(rel, "./"))
	for _, p := range g.patterns {
		if p.matches(rel, isDir) {
			return true
		}
	}
	return false
}

func (p gitignorePattern) matches(rel string, isDir bool) bool {
	if p.dirOnly && !isDir && rel != p.pattern && !strings.HasPrefix(rel, p.pattern+"/") {
		return false
	}
	if p.anchored || strings.Contains(p.pattern, "/") {
		if rel == p.pattern || strings.HasPrefix(rel, p.pattern+"/") {
			return true
		}
		ok, _ := filepath.Match(p.pattern, rel)
		return ok
	}
	base := filepath.Base(rel)
	if base == p.pattern {
		return true
	}
	ok, _ := filepath.Match(p.pattern, base)
	return ok
}

// addZipBytes writes a synthetic entry, for content that belongs in the archive
// without belonging in the working tree.
func addZipBytes(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func addZipFile(zw *zip.Writer, rootAbs, pathAbs string) error {
	st, err := os.Stat(pathAbs)
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(st)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(rel)
	header.Method = zip.Deflate
	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	in, err := os.Open(pathAbs)
	if err != nil {
		return err
	}
	defer in.Close()
	_, err = io.Copy(writer, in)
	return err
}

func fileSHA256(path string) (string, error) {
	in, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer in.Close()
	h := sha256.New()
	if _, err := io.Copy(h, in); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func shouldSkipPackDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", ".hg", ".svn", "node_modules", "__pycache__", ".pytest_cache", ".venv", "venv", "dist":
		return true
	}
	return false
}

func shouldSkipPackFile(name string) bool {
	switch strings.ToLower(name) {
	case ".ds_store", ".git":
		return true
	}
	return false
}
