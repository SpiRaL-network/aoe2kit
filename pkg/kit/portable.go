package kit

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type PortableReport struct {
	Root     string          `json:"root"`
	Profile  string          `json:"profile,omitempty"`
	OK       bool            `json:"ok"`
	Checks   []PortableCheck `json:"checks"`
	Warnings []string        `json:"warnings,omitempty"`
	Errors   []string        `json:"errors,omitempty"`
}

type PortableCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

type PortableCheckOptions struct {
	Profile PackProfile
}

func PortableCheckRoot(root string) (PortableReport, error) {
	return PortableCheckRootWithOptions(root, PortableCheckOptions{Profile: ProfileHandoff})
}

func PortableCheckRootWithOptions(root string, opts PortableCheckOptions) (PortableReport, error) {
	if opts.Profile == "" {
		opts.Profile = ProfileHandoff
	}
	report := PortableReport{Root: root, Profile: string(opts.Profile)}
	add := func(name string, ok bool, detail string) {
		report.Checks = append(report.Checks, PortableCheck{Name: name, OK: ok, Detail: detail})
		if !ok {
			report.Errors = append(report.Errors, detail)
		}
	}
	if !KnownProfile(opts.Profile) {
		add("profile", false, fmt.Sprintf("unknown profile %q", opts.Profile))
		report.OK = false
		return report, nil
	}
	var publicAllowed map[string]bool
	if opts.Profile == ProfilePublic {
		var allowErr error
		publicAllowed, allowErr = readPublicAllowlist(root)
		if allowErr != nil {
			add("public_allowlist", false, allowErr.Error())
		} else {
			leaks, walkErr := publicAllowlistLeaks(root, publicAllowed)
			if walkErr != nil {
				add("public_allowlist", false, "public allowlist scan failed: "+walkErr.Error())
			} else {
				detail := "exact allowlist"
				if len(leaks) > 0 {
					detail = fmt.Sprintf("unlisted files: %s", strings.Join(leaks, ", "))
				}
				add("public_allowlist", len(leaks) == 0, detail)
			}
		}
	}
	verify := Verify(root, false)
	add("kit_verify", verify.OK(), "kit verify structural gate")
	add("source_build", verify.SourceBuildOK, "go build ./...")
	if opts.Profile.includesBinary() {
		binaryPath := filepath.Join(root, "kit")
		if st, err := os.Stat(binaryPath); err == nil && !st.IsDir() {
			hash, hashErr := fileSHA256(binaryPath)
			if hashErr != nil {
				add("bundled_linux_amd64_binary", false, "hash bundled kit: "+hashErr.Error())
			} else {
				add("bundled_linux_amd64_binary", true, "present sha256="+hash)
			}
		} else if err != nil {
			add("bundled_linux_amd64_binary", false, "missing bundled linux-amd64 kit binary")
		} else {
			add("bundled_linux_amd64_binary", false, "kit exists but is a directory")
		}
	} else {
		add("bundled_linux_amd64_binary", true, fmt.Sprintf("excluded by %s profile", opts.Profile))
	}
	for _, errMsg := range verify.Errors {
		report.Errors = append(report.Errors, errMsg)
	}
	required, reqErr := RequiredDocsIn(root, opts.Profile)
	if reqErr != nil {
		add("required_docs", false, reqErr.Error())
	}
	for _, rel := range required {
		_, statErr := os.Stat(filepath.Join(root, rel))
		detail := "present"
		if statErr != nil {
			detail = "missing required portable artifact " + rel
		}
		add("required:"+rel, statErr == nil, detail)
	}
	var scripts []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("walk %s: %v", path, err))
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if excluded(rel, opts.Profile.excludes()) {
			return nil
		}
		low := strings.ToLower(rel)
		if strings.HasSuffix(low, ".py") || strings.HasSuffix(low, ".sh") {
			scripts = append(scripts, rel)
		}
		return nil
	})
	if err != nil {
		return report, err
	}
	if opts.Profile == ProfilePublic {
		binaryLeaks, binaryErr := portableExecutableLeaksForProfile(root, opts.Profile)
		if binaryErr != nil {
			add("third_party_binaries", false, "public binary scan failed: "+binaryErr.Error())
		} else {
			detail := "none"
			if len(binaryLeaks) > 0 {
				detail = fmt.Sprintf("executable or binary payloads: %s", strings.Join(binaryLeaks, ", "))
			}
			add("third_party_binaries", len(binaryLeaks) == 0, detail)
		}
	}
	pathLeaks, pathLeakErr := portableLocalPathLeaksForProfile(root, opts.Profile)
	if pathLeakErr != nil {
		add("local_path_leaks", false, "local path leak scan failed: "+pathLeakErr.Error())
	} else {
		leakDetail := "none"
		if len(pathLeaks) > 0 {
			leakDetail = fmt.Sprintf("local path leaks: %s", strings.Join(pathLeaks, ", "))
		}
		add("local_path_leaks", len(pathLeaks) == 0, leakDetail)
	}
	excludedRefs, excludedErr := portableExcludedLocalPathReferenceCountForProfile(root, opts.Profile)
	if excludedErr != nil {
		report.Warnings = append(report.Warnings, "non-shipped local path reference scan failed: "+excludedErr.Error())
	} else if excludedRefs > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("%d local path references in non-shipped files", excludedRefs))
	}
	identityLeaks, identityErr := portableIdentityLeaksForProfile(root, opts.Profile)
	if identityErr != nil {
		add("identity_leaks", false, "identity leak scan failed: "+identityErr.Error())
	} else {
		identityDetail := "none"
		if len(identityLeaks) > 0 {
			identityDetail = fmt.Sprintf("identity leaks: %s", strings.Join(identityLeaks, ", "))
		}
		add("identity_leaks", len(identityLeaks) == 0, identityDetail)
	}
	if len(scripts) > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("script files present; durable workflows should be Go tools or documented gaps: %s", strings.Join(scripts, ", ")))
	}
	report.OK = len(report.Errors) == 0
	return report, nil
}

func portableHandoffFiles(root string) (string, []string, error) {
	return portableProfileFiles(root, ProfileHandoff)
}

func portableProfileFiles(root string, profile PackProfile) (string, []string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", nil, err
	}
	files, err := packFileListForProfile(rootAbs, filepath.Join(rootAbs, ".aoe2kit-portable-check-output.zip"), profile, profile.excludes(), profile.includesBinary())
	if err != nil {
		return "", nil, err
	}
	return rootAbs, files, nil
}

func publicAllowlistLeaks(root string, allowed map[string]bool) ([]string, error) {
	ignore, err := loadRootGitignore(root)
	if err != nil {
		return nil, err
	}
	var leaks []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if rel == ".git" || rel == "build" || excluded(rel, []string{"tools/**"}) {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == ProfileMarkerName || strings.HasSuffix(rel, ".aoe2kit-portable-check-output.zip") || ignore.ignored(rel, false) {
			return nil
		}
		if !allowed[rel] {
			leaks = append(leaks, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(leaks)
	return leaks, nil
}

// portableExecutableLeaksForProfile protects the public source-only profile
// from accidentally shipping a tool executable or vendored binary. The
// profile file list is the authority here, so excluded diagnostics cannot
// create a false release failure.
func portableExecutableLeaksForProfile(root string, profile PackProfile) ([]string, error) {
	rootAbs, files, err := portableProfileFiles(root, profile)
	if err != nil {
		return nil, err
	}
	var leaks []string
	for _, path := range files {
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.Mode().Perm()&0o111 != 0 {
			leaks = append(leaks, rel+" (executable mode)")
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if isExecutableMagic(data) || strings.Contains(strings.ToLower(filepath.Base(rel)), "xs-check") {
			leaks = append(leaks, rel+" (binary or executable payload)")
		}
	}
	return leaks, nil
}

func isExecutableMagic(data []byte) bool {
	if len(data) >= 4 && data[0] == 0x7f && data[1] == 'E' && data[2] == 'L' && data[3] == 'F' {
		return true
	}
	if len(data) >= 2 && data[0] == 'M' && data[1] == 'Z' {
		return true
	}
	if len(data) >= 4 {
		magic := string(data[:4])
		return magic == "\xfe\xed\xfa\xce" || magic == "\xce\xfa\xed\xfe" || magic == "\xfe\xed\xfa\xcf" || magic == "\xcf\xfa\xed\xfe"
	}
	return false
}

func portableLocalPathLeaks(root string) ([]string, error) {
	return portableLocalPathLeaksForProfile(root, ProfileHandoff)
}

func portableLocalPathLeaksForProfile(root string, profile PackProfile) ([]string, error) {
	rootAbs, files, err := portableProfileFiles(root, profile)
	if err != nil {
		return nil, err
	}
	return portableLocalPathLeaksInFiles(rootAbs, files)
}

func portableLocalPathLeaksInFiles(rootAbs string, files []string) ([]string, error) {
	var leaks []string
	for _, path := range files {
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil || looksBinary(data) {
			continue
		}
		for _, needle := range portableLocalPathNeedles() {
			if bytes.Contains(data, []byte(needle)) {
				leaks = append(leaks, rel+": "+needle)
			}
		}
	}
	return leaks, nil
}

func portableExcludedLocalPathReferenceCount(root string) (int, error) {
	return portableExcludedLocalPathReferenceCountForProfile(root, ProfileHandoff)
}

func portableExcludedLocalPathReferenceCountForProfile(root string, profile PackProfile) (int, error) {
	rootAbs, shippedFiles, err := portableProfileFiles(root, profile)
	if err != nil {
		return 0, err
	}
	shipped := make(map[string]bool, len(shippedFiles))
	for _, path := range shippedFiles {
		abs, err := filepath.Abs(path)
		if err != nil {
			return 0, err
		}
		shipped[abs] = true
	}
	count := 0
	err = filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		if shipped[abs] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil || looksBinary(data) {
			return nil
		}
		for _, needle := range portableLocalPathNeedles() {
			count += bytes.Count(data, []byte(needle))
		}
		return nil
	})
	return count, err
}

func portableLocalPathNeedles() []string {
	needles := []string{"C:" + "\\Users\\", "/mnt/c/" + "Users/"}
	if home, homeErr := os.UserHomeDir(); homeErr == nil && len(home) > 1 {
		needles = append(needles, strings.TrimSuffix(home, "/")+"/")
	}
	return needles
}

func looksBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	limit := len(data)
	if limit > 4096 {
		limit = 4096
	}
	return bytes.IndexByte(data[:limit], 0) >= 0
}

var portableIdentityNameRe = regexp.MustCompile(`\b` + "Ry" + `an\b`)

func portableIdentityLeaks(root string) ([]string, error) {
	return portableIdentityLeaksForProfile(root, ProfileHandoff)
}

func portableIdentityLeaksForProfile(root string, profile PackProfile) ([]string, error) {
	rootAbs, files, err := portableProfileFiles(root, profile)
	if err != nil {
		return nil, err
	}
	var leaks []string
	for _, path := range files {
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		if !portableIdentityScanFile(rel, profile) {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil || looksBinary(data) {
			continue
		}
		leaks = append(leaks, portableIdentityLeaksInText(rel, data)...)
	}
	return leaks, nil
}

func portableIdentityScanFile(rel string, profile PackProfile) bool {
	if rel == "LICENSE" {
		return false
	}
	if excluded(rel, profile.excludes()) {
		return false
	}
	ext := strings.ToLower(filepath.Ext(rel))
	switch ext {
	case ".go", ".md", ".json", ".txt", ".xs", ".per", ".ai", ".toml", ".yaml", ".yml":
		return true
	}
	if strings.HasPrefix(rel, "data/") && ext == ".json" {
		return true
	}
	switch rel {
	case "README", "Makefile", "go.mod", "go.sum":
		return true
	}
	return false
}

func portableIdentityLeaksInText(rel string, data []byte) []string {
	lines := strings.Split(string(data), "\n")
	var leaks []string
	for i, line := range lines {
		if match := portableIdentityNameRe.FindString(line); match != "" {
			leaks = append(leaks, fmt.Sprintf("%s:%d: %s", rel, i+1, match))
		}
	}
	return leaks
}
