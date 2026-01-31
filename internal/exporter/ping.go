package exporter

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/go-ping/ping"
)

func (e *Exporter) pingLeases(ctx context.Context, leases []lease) {
	sem := make(chan struct{}, e.pingWorkers)
	var wg sync.WaitGroup
	var successes, failures int
	start := time.Now()

	now := time.Now()
	valid := make([]lease, 0, len(leases))
	for _, l := range leases {
		if now.After(l.expiry) {
			continue
		}
		valid = append(valid, l)
	}
	if len(valid) == 0 {
		return
	}

	for _, l := range valid {
		l := l
		wg.Add(1)
		go func() {
			defer wg.Done()
			delay := e.randomDelay()
			if delay > 0 {
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return
				}
			}
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			ok := pingHost(l.ip, e.pingTimeout, e.logAllPings)
			if ok {
				e.recordSeen(l.ip)
				successes++
			} else {
				failures++
			}
		}()
	}

	wg.Wait()
	log.Printf("ping cycle done leases=%d successes=%d failures=%d dur=%s", len(valid), successes, failures, time.Since(start))
}

func pingHost(ip string, timeout time.Duration, logAll bool) bool {
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
	success := stats.PacketsRecv > 0
	if logAll {
		if success {
			log.Printf("ping success %s rtt=%s", ip, stats.AvgRtt)
		} else {
			log.Printf("ping failure %s", ip)
		}
	} else if !success {
		log.Printf("ping failure %s", ip)
	}
	return success
}
