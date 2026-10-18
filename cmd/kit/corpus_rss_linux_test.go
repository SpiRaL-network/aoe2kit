package main

import (
	"strings"
	"testing"
)

func TestLinuxReaderVmHWM(t *testing.T) {
	for _, tc := range []struct {
		text string
		want int64
	}{
		{"VmPeak: 999999 kB\nVmHWM:\t12345 kB\nVmRSS: 10000 kB\n", 12345},
		{"VmRSS: 10000 kB\n", 0}, {"VmHWM: -1 kB\n", 0}, {"VmHWM: 25 MB\n", 0},
	} {
		if n := readLinuxVmHWM(strings.NewReader(tc.text)); n != tc.want {
			t.Fatalf("got %d want %d", n, tc.want)
		}
	}
}
