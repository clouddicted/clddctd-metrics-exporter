package exporter

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/go-ping/ping"
)

func (e *Exporter) pingLeases(ctx context.Context, leases []lease) {
	if len(leases) == 0 {
		return
	}
	sem := make(chan struct{}, e.pingWorkers)
	var wg sync.WaitGroup

	for _, l := range leases {
		l := l
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			if pingHost(l.ip, e.pingTimeout) {
				e.recordSeen(l.ip)
			}
		}()
	}

	wg.Wait()
}

func pingHost(ip string, timeout time.Duration) bool {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		log.Printf("ping setup failed for %s: %v", ip, err)
		return false
	}
	pinger.SetPrivileged(true)
	pinger.Count = 1
	pinger.Timeout = timeout
	if err := pinger.Run(); err != nil {
		log.Printf("ping run failed for %s: %v", ip, err)
		return false
	}
	stats := pinger.Statistics()
	return stats.PacketsRecv > 0
}
