package replay

import (
	"encoding/json"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// TrackMemory samples Go allocation counters. Linux VmHWM is reported separately:
// runtime.Sys is not RSS, and sampling can miss brief heap peaks.
func TrackMemory(out io.Writer) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	var heap, sys uint64
	sample := func() {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		heap = max(heap, m.HeapAlloc)
		sys = max(sys, m.Sys)
	}
	sample()
	go func() {
		defer close(done)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				sample()
			case <-stop:
				sample()
				return
			}
		}
	}()
	return func() {
		close(stop)
		<-done
		report := struct {
			Heap    uint64 `json:"sampled_peak_heap_bytes"`
			Sys     uint64 `json:"sampled_peak_runtime_sys_bytes"`
			RSS     uint64 `json:"process_peak_rss_bytes,omitempty"`
			Target  uint64 `json:"target_rss_bytes"`
			Workers int    `json:"workers"`
		}{Heap: heap, Sys: sys, Target: 300 << 20, Workers: 1}
		if raw, err := os.ReadFile("/proc/self/status"); err == nil {
			for _, line := range strings.Split(string(raw), "\n") {
				fields := strings.Fields(line)
				if len(fields) == 3 && fields[0] == "VmHWM:" {
					kb, _ := strconv.ParseUint(fields[1], 10, 64)
					report.RSS = kb * 1024
				}
			}
		}
		_ = json.NewEncoder(out).Encode(report)
	}
}
