package aifile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Reference struct {
	Raw       string `json:"raw"`
	Source    string `json:"source,omitempty"`
	Domain    string `json:"domain,omitempty"`
	File      string `json:"file"`
	BaseName  string `json:"base_name"`
	Extension string `json:"extension"`
	SourceMod string `json:"source_mod,omitempty"`
	Flag      string `json:"flag,omitempty"`
}

type Group struct {
	Name      string   `json:"name"`
	SourceMod string   `json:"source_mod,omitempty"`
	Files     []string `json:"files"`
	Raw       []string `json:"raw"`
}

type Loadout struct {
	References []Reference `json:"references"`
	Groups     []Group     `json:"groups"`
	Signature  string      `json:"signature"`
	Empty      bool        `json:"empty"`
	Note       string      `json:"note,omitempty"`
}

type Fingerprint struct {
	Path        string            `json:"path"`
	SHA256      string            `json:"sha256"`
	Method      string            `json:"method"`
	Files       []FingerprintFile `json:"files"`
	RegistryKey string            `json:"registry_key,omitempty"`
	Known       *RegistryEntry    `json:"known,omitempty"`
}

type FingerprintFile struct {
	Path            string `json:"path"`
	RelativePath    string `json:"relative_path"`
	SizeBytes       int64  `json:"size_bytes"`
	NormalizedBytes int    `json:"normalized_bytes"`
	ContentSHA256   string `json:"content_sha256"`
	LineEndingsNote string `json:"line_endings_note,omitempty"`
}

type RegistryEntry struct {
	Name          string   `json:"name"`
	Mod           string   `json:"mod,omitempty"`
	Notes         string   `json:"notes,omitempty"`
	Signature     string   `json:"signature,omitempty"`
	Files         []string `json:"files,omitempty"`
	ContentSHA256 string   `json:"content_sha256,omitempty"`
}

type Registry map[string]RegistryEntry

func ParseReferences(stringsIn []string) []Reference {
	seen := map[string]bool{}
	var refs []Reference
	for _, raw := range stringsIn {
		raw = strings.TrimSpace(raw)
		if raw == "" || !strings.Contains(raw, ":AI:") {
			continue
		}
		ref, ok := ParseReference(raw)
		if !ok {
			continue
		}
		key := ref.Normalized()
		if seen[key] {
			continue
		}
		seen[key] = true
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Normalized() < refs[j].Normalized() })
	return refs
}

func ParseReference(raw string) (Reference, bool) {
	parts := strings.Split(raw, ":")
	if len(parts) < 5 {
		return Reference{}, false
	}
	source := parts[0]
	domain := parts[1]
	file := parts[2]
	if !strings.EqualFold(domain, "AI") {
		return Reference{}, false
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(file), "."))
	if ext != "ai" && ext != "per" && ext != "per2" {
		return Reference{}, false
	}
	return Reference{
		Raw:       raw,
		Source:    source,
		Domain:    domain,
		File:      file,
		BaseName:  strings.TrimSuffix(file, filepath.Ext(file)),
		Extension: ext,
		SourceMod: parts[3],
		Flag:      parts[4],
	}, true
}

func (r Reference) Normalized() string {
	return strings.Join([]string{
		strings.ToUpper(r.Source),
		strings.ToUpper(r.Domain),
		r.File,
		r.SourceMod,
		strings.ToLower(r.Flag),
	}, ":")
}

func NewLoadout(refs []Reference) Loadout {
	if refs == nil {
		refs = []Reference{}
	}
	groups := GroupReferences(refs)
	if groups == nil {
		groups = []Group{}
	}
	loadout := Loadout{
		References: refs,
		Groups:     groups,
		Signature:  Signature(refs),
		Empty:      len(refs) == 0,
	}
	if loadout.Empty {
		loadout.Note = "empty ai_files: replay names no custom AI files; AI slots use the game's selected/default AI, not embedded script content"
	} else {
		loadout.Note = "replay names custom AI files only; script content must be fingerprinted from .ai/.per files"
	}
	return loadout
}

func GroupReferences(refs []Reference) []Group {
	byKey := map[string]*Group{}
	var keys []string
	for _, ref := range refs {
		key := ref.SourceMod + "\x00" + ref.BaseName
		group := byKey[key]
		if group == nil {
			group = &Group{Name: ref.BaseName, SourceMod: ref.SourceMod}
			byKey[key] = group
			keys = append(keys, key)
		}
		group.Files = append(group.Files, ref.File)
		group.Raw = append(group.Raw, ref.Raw)
	}
	sort.Strings(keys)
	out := make([]Group, 0, len(keys))
	for _, key := range keys {
		group := *byKey[key]
		sort.Strings(group.Files)
		sort.Strings(group.Raw)
		out = append(out, group)
	}
	return out
}

func Signature(refs []Reference) string {
	lines := make([]string, 0, len(refs))
	for _, ref := range refs {
		lines = append(lines, ref.Normalized())
	}
	sort.Strings(lines)
	var buf bytes.Buffer
	buf.WriteString("aoe2kit:ai_files:v1\n")
	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:])
}

func FingerprintPath(path string) (*Fingerprint, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	var files []string
	root := path
	if st.IsDir() {
		err := filepath.WalkDir(path, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if IsAIPath(p) {
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		if !IsAIPath(path) {
			return nil, fmt.Errorf("%s is not an .ai/.per file", path)
		}
		root = filepath.Dir(path)
		files = append(files, path)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s contains no .ai/.per files", path)
	}
	sort.Strings(files)
	var payload bytes.Buffer
	payload.WriteString("aoe2kit:ai_content:v1\n")
	out := &Fingerprint{Path: path, Method: "sorted AI files with CRLF/CR normalized to LF"}
	for _, file := range files {
		st, err := os.Stat(file)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		normalized, note := normalizeText(data)
		fileSum := sha256.Sum256(normalized)
		rel, err := filepath.Rel(root, file)
		if err != nil {
			rel = filepath.Base(file)
		}
		rel = filepath.ToSlash(rel)
		payload.WriteString("file:")
		payload.WriteString(rel)
		payload.WriteByte('\n')
		payload.WriteString("sha256:")
		payload.WriteString(hex.EncodeToString(fileSum[:]))
		payload.WriteByte('\n')
		payload.WriteString("bytes:")
		payload.WriteString(fmt.Sprintf("%d", len(normalized)))
		payload.WriteByte('\n')
		out.Files = append(out.Files, FingerprintFile{
			Path:            file,
			RelativePath:    rel,
			SizeBytes:       st.Size(),
			NormalizedBytes: len(normalized),
			ContentSHA256:   hex.EncodeToString(fileSum[:]),
			LineEndingsNote: note,
		})
	}
	sum := sha256.Sum256(payload.Bytes())
	out.SHA256 = hex.EncodeToString(sum[:])
	out.RegistryKey = "ai_content:" + out.SHA256
	return out, nil
}

func IsAIPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ai", ".per", ".per2":
		return true
	default:
		return false
	}
}

func normalizeText(data []byte) ([]byte, string) {
	note := ""
	if bytes.Contains(data, []byte("\r\n")) || bytes.Contains(data, []byte("\r")) {
		note = "line endings normalized"
	}
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
	return data, note
}

func LoadRegistry(path string) (Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Registry{}, nil
		}
		return nil, err
	}
	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, err
	}
	if registry == nil {
		registry = Registry{}
	}
	return registry, nil
}

func SaveRegistry(path string, registry Registry) error {
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
