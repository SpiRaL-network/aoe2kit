package aoe2

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FileInfo struct {
	Path      string `json:"path"`
	Base      string `json:"base"`
	Kind      string `json:"kind"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

func InspectFile(path string) (FileInfo, error) {
	st, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}
	if st.IsDir() {
		return FileInfo{}, fmt.Errorf("%s is a directory", path)
	}
	hash, err := sha256File(path)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{
		Path:      path,
		Base:      filepath.Base(path),
		Kind:      KindForPath(path),
		SizeBytes: st.Size(),
		SHA256:    hash,
	}, nil
}

func KindForPath(path string) string {
	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(base))
	switch ext {
	case ".aoe2scenario":
		return "scenario"
	case ".aoe2record":
		return "record"
	case ".dat":
		return "dat"
	case ".ai", ".per", ".per2":
		return "ai"
	case ".sld":
		return "gfx"
	case ".json":
		return "json"
	case ".txt":
		return "text"
	case ".zip":
		return "zip"
	}
	return strings.TrimPrefix(ext, ".")
}

func sha256File(path string) (string, error) {
	fh, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fh.Close()

	h := sha256.New()
	if _, err := io.Copy(h, fh); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
