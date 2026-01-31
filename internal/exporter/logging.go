package exporter

import (
	"log"
	"os"
	"sync"
)

var (
	hostOnce   sync.Once
	hostCached string
)

func hostname() string {
	hostOnce.Do(func() {
		h, err := os.Hostname()
		if err != nil || h == "" {
			h = "unknown"
		}
		hostCached = h
	})
	return hostCached
}

// Logf emits logfmt-style lines with level and host.
func Logf(level string, format string, args ...interface{}) {
	prefix := "level=%s host=%s "
	allArgs := append([]interface{}{level, hostname()}, args...)
	log.Printf(prefix+format, allArgs...)
}
