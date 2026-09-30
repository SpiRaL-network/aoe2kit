package cba

import (
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Per-worker memory budgets, in MiB, from measured peak RSS of a single replay:
//
//	extraction (BuildPerformance): ~860 MiB  -- inflated replay + profile + events +
//	                                            camera + checksum series + razes
//	classification (registry):     ~207 MiB  -- inflated replay + player profile only
//
// Budgets sit above the measurement because replay size varies and Go returns freed
// memory to the OS lazily.
const (
	ExtractWorkerMiB  = 1100
	RegistryWorkerMiB = 300
)

// workersFor picks a concurrency level that fits in memory.
//
// Concurrency here is bounded by RAM, not by CPU. Defaulting to NumCPU cost this machine
// its swap: 8 extraction workers at ~860 MiB each is ~6.9 GiB on a 7.5 GiB box. CPU count
// says nothing about whether the work FITS.
//
// requested > 0 is honoured as-is -- an explicit --workers is the operator overriding the
// heuristic deliberately, and this must stay usable on a big machine.
func workersFor(perWorkerMiB, requested int) int {
	if requested > 0 {
		return requested
	}

	// NumCPU is NOT trustworthy on shared hosting: a box may advertise 128 processors
	// while our affinity allows 2, and MemAvailable reports the whole machine's memory
	// rather than our slice -- so the memory guard never binds either. Believe the
	// smaller of the two CPU signals.
	n := usableCPUs()
	if avail := availableMiB(); avail > 0 {
		// Leave most of the machine alone: this is a batch job sharing a desktop with
		// the user's actual work, and pushing into swap is far worse than running slower.
		budget := avail / 2
		fit := budget / perWorkerMiB
		if fit < n {
			n = fit
		}
	} else {
		// Unknown memory: assume a small machine rather than a large one. Guessing high
		// costs a swap storm; guessing low costs some wall-clock.
		n = 2
	}

	if n < 1 {
		n = 1
	}
	return n
}

// usableCPUs returns how many CPUs we may actually run on.
//
// runtime.NumCPU can report the host's full processor count on shared hosting even when
// scheduler affinity restricts us to a couple of cores. Running one heavy worker per
// advertised CPU then thrashes: on DreamHost that meant 10 concurrent ~1 GiB extractions
// on 2 real cores, burning 32x the CPU for the same work. Cross-check against the
// affinity mask and take the smaller.
func usableCPUs() int {
	n := runtime.NumCPU()
	if aff := affinityCPUs(); aff > 0 && aff < n {
		return aff
	}
	return n
}

var cpusAllowedRe = regexp.MustCompile(`Cpus_allowed_list:\s*(\S+)`)

// affinityCPUs parses /proc/self/status Cpus_allowed_list ("0-1", "0,3-5"), or 0 if
// unavailable.
func affinityCPUs() int {
	raw, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	m := cpusAllowedRe.FindSubmatch(raw)
	if m == nil {
		return 0
	}
	total := 0
	for _, part := range strings.Split(string(m[1]), ",") {
		lo, hi, found := strings.Cut(part, "-")
		a, err := strconv.Atoi(strings.TrimSpace(lo))
		if err != nil {
			continue
		}
		if !found {
			total++
			continue
		}
		b, err := strconv.Atoi(strings.TrimSpace(hi))
		if err != nil {
			continue
		}
		if b >= a {
			total += b - a + 1
		}
	}
	return total
}

var memAvailableRe = regexp.MustCompile(`MemAvailable:\s+(\d+) kB`)

// availableMiB reports memory available without swapping, or 0 if unknown.
// MemAvailable (not MemFree) is the right figure -- it accounts for reclaimable cache.
func availableMiB() int {
	raw, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	m := memAvailableRe.FindSubmatch(raw)
	if m == nil {
		return 0
	}
	kb, err := strconv.Atoi(string(m[1]))
	if err != nil {
		return 0
	}
	return kb / 1024
}
