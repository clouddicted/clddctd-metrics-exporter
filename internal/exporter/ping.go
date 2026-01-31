package exporter

import (
	"context"
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
				e.recordSeen(l.ip, time.Now())
				successes++
			} else {
				failures++
			}
		}()
	}

	wg.Wait()
	Logf("info", "msg=\"ping cycle done\" leases=%d successes=%d failures=%d dur_ms=%d", len(valid), successes, failures, time.Since(start).Milliseconds())
}

func pingHost(ip string, timeout time.Duration, logAll bool) bool {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		Logf("error", "msg=\"ping setup failed\" ip=%s err=%v", ip, err)
		return false
	}
	pinger.SetPrivileged(true)
	pinger.Count = 1
	pinger.Timeout = timeout
	if err := pinger.Run(); err != nil {
		Logf("error", "msg=\"ping run failed\" ip=%s err=%v", ip, err)
		return false
	}
	stats := pinger.Statistics()
	success := stats.PacketsRecv > 0
	if logAll {
		if success {
			Logf("info", "msg=\"ping success\" ip=%s rtt_ms=%d", ip, stats.AvgRtt.Milliseconds())
		} else {
			Logf("error", "msg=\"ping failure\" ip=%s", ip)
		}
	} else if !success {
		Logf("error", "msg=\"ping failure\" ip=%s", ip)
	}
	return success
}
