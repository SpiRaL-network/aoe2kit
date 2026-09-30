package replay

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"strings"
	"testing"
)

func TestSyncBudgetBeforeAllocation(t *testing.T) {
	var prefix [8]byte
	binary.LittleEndian.PutUint32(prefix[:4], 8)
	err := checkSyncReaderBudget(bytes.NewReader(prefix[:]), 8+(512<<20)/256+1)
	if err == nil || !strings.Contains(err.Error(), "body bytes") {
		t.Fatal(err)
	}
}

func TestSyncBudgetInflationLimit(t *testing.T) {
	var compressed bytes.Buffer
	z, _ := flate.NewWriter(&compressed, flate.BestSpeed)
	var zeros [4096]byte
	for i := 0; i < (64<<20)/len(zeros)+1; i++ {
		if _, e := z.Write(zeros[:]); e != nil {
			t.Fatal(e)
		}
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	data := make([]byte, 8+compressed.Len())
	binary.LittleEndian.PutUint32(data[:4], uint32(len(data)))
	copy(data[8:], compressed.Bytes())
	err := checkSyncReaderBudget(bytes.NewReader(data), uint64(len(data)))
	if err == nil || !strings.Contains(err.Error(), "inflated header") {
		t.Fatal(err)
	}
}
