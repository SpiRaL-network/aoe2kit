//go:build !linux

package main

func selfPeakRSS() int64 { return 0 }
