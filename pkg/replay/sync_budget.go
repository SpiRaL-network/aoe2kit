package replay

import (
	"archive/zip"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// checkSyncBudget runs before eager input/header/event allocation, including
// for zipped inputs. The body multiplier is a conservative guardrail, not a
// predicted RSS: full SyncEvents carry matrices, annotations and delta models.
func checkSyncBudget(path string) error {
	if strings.EqualFold(filepath.Ext(path), ".zip") {
		z, err := zip.OpenReader(path)
		if err != nil {
			return err
		}
		defer z.Close()
		for _, f := range z.File {
			if !strings.EqualFold(filepath.Ext(f.Name), ".aoe2record") {
				continue
			}
			r, err := f.Open()
			if err != nil {
				return err
			}
			defer r.Close()
			return checkSyncReaderBudget(r, f.UncompressedSize64)
		}
		return fmt.Errorf("zip contains no .aoe2record")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	s, err := f.Stat()
	if err != nil {
		return err
	}
	return checkSyncReaderBudget(f, uint64(s.Size()))
}

func checkSyncReaderBudget(r io.Reader, size uint64) error {
	var prefix [8]byte
	if _, err := io.ReadFull(r, prefix[:]); err != nil {
		return err
	}
	header := uint64(binary.LittleEndian.Uint32(prefix[:4]))
	if header < 8 || header > size {
		return fmt.Errorf("invalid header length %d", header)
	}
	body := size - header
	if body > (512<<20)/256 {
		return fmt.Errorf("sync memory guard: %d body bytes exceed the conservative 512 MiB materialization budget (256x body estimate); use kit replay health on the unzipped recording for bounded triage", body)
	}
	z := flate.NewReader(io.LimitReader(r, int64(header-8)))
	defer z.Close()
	n, err := io.Copy(io.Discard, io.LimitReader(z, (64<<20)+1))
	if err != nil {
		return fmt.Errorf("header inflate: %w", err)
	}
	if n > 64<<20 {
		return fmt.Errorf("sync memory guard: inflated header exceeds 64 MiB before eager parsing; use kit replay health on the unzipped recording")
	}
	return nil
}
