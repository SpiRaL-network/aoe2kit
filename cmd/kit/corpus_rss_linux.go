package main

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
)

// VmHWM belongs to the current post-exec address space. wait4/rusage may also
// include the forked parent's pre-exec footprint, which is not reader memory.
func selfPeakRSS() int64 {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0
	}
	defer f.Close()
	return readLinuxVmHWM(f)
}

func readLinuxVmHWM(r io.Reader) int64 {
	s := bufio.NewScanner(r)
	for s.Scan() {
		parts := strings.Fields(s.Text())
		if len(parts) == 3 && parts[0] == "VmHWM:" && parts[2] == "kB" {
			n, e := strconv.ParseInt(parts[1], 10, 64)
			if e == nil && n > 0 {
				return n
			}
			return 0
		}
	}
	return 0
}
