package exporter

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Exporter struct {
	leaseFile       string
	wanInterface    string
	onlineThreshold time.Duration
	pingTimeout     time.Duration
	pingWorkers     int
	pingCycleMax    time.Duration
	logAllPings     bool

	mu       sync.Mutex
	lastSeen map[string]time.Time // keyed by IP
	randMu   sync.Mutex
	rnd      *rand.Rand

	up            prometheus.Gauge
	dhcpEnabled   prometheus.Gauge
	natEnabled    prometheus.Gauge
	leaseInfo     *prometheus.GaugeVec
	leaseOnline   *prometheus.GaugeVec
	leaseLastSeen *prometheus.GaugeVec
}

const (
	exporterNamespace = "gateway"
)

func NewExporter(leaseFile, wanInterface string, onlineThreshold, pingTimeout time.Duration, pingWorkers int, pingCycleMax time.Duration, logAllPings bool) *Exporter {
	if pingCycleMax <= 0 {
		pingCycleMax = 5 * time.Second
	}
	return &Exporter{
		leaseFile:       leaseFile,
		wanInterface:    wanInterface,
		onlineThreshold: onlineThreshold,
		pingTimeout:     pingTimeout,
		pingWorkers:     pingWorkers,
		pingCycleMax:    pingCycleMax,
		logAllPings:     logAllPings,
		lastSeen:        make(map[string]time.Time),
		rnd:             rand.New(rand.NewSource(time.Now().UnixNano())),
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: exporterNamespace,
			Name:      "up",
			Help:      "Gateway exporter scrape success (1) or failure (0).",
		}),
		dhcpEnabled: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: exporterNamespace,
			Name:      "dhcp_enabled",
			Help:      "Whether dnsmasq DHCP service is running (1) or not (0).",
		}),
		natEnabled: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: exporterNamespace,
			Name:      "nat_enabled",
			Help:      "Whether NAT is enabled (ip_forward and MASQUERADE rule present).",
		}),
		leaseInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: exporterNamespace,
			Name:      "dhcp_lease_info",
			Help:      "Static info for each DHCP lease (always 1).",
		}, []string{"mac", "host", "ip"}),
		leaseOnline: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: exporterNamespace,
			Name:      "dhcp_lease_online",
			Help:      "Whether the lease responded to ping within online window (1/0).",
		}, []string{"mac", "host", "ip"}),
		leaseLastSeen: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: exporterNamespace,
			Name:      "dhcp_lease_last_seen_seconds",
			Help:      "Seconds since the lease last responded to ping (-1 if never).",
		}, []string{"mac", "host", "ip"}),
	}
}

// Start runs continuous background ping cycles.
func (e *Exporter) Start(ctx context.Context) {
	for {
		start := time.Now()
		e.backgroundPing(ctx)
		elapsed := time.Since(start)
		sleepFor := e.pingCycleMax - elapsed
		if sleepFor > 0 {
			select {
			case <-time.After(sleepFor):
			case <-ctx.Done():
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (e *Exporter) randomDelay() time.Duration {
	max := e.pingCycleMax
	if max <= 0 {
		return 0
	}
	e.randMu.Lock()
	defer e.randMu.Unlock()
	return time.Duration(e.rnd.Int63n(max.Nanoseconds()))
}

func (e *Exporter) backgroundPing(ctx context.Context) {
	leases, err := readLeases(e.leaseFile)
	if err != nil {
		log.Printf("background lease read failed: %v", err)
		return
	}
	e.pingLeases(ctx, leases)
}

// Describe implements the Collector interface.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.up.Describe(ch)
	e.dhcpEnabled.Describe(ch)
	e.natEnabled.Describe(ch)
	e.leaseInfo.Describe(ch)
	e.leaseOnline.Describe(ch)
	e.leaseLastSeen.Describe(ch)
}

// Collect runs a scrape and exports metrics.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	if err := e.scrape(); err != nil {
		log.Printf("scrape error: %v", err)
		e.up.Set(0)
	} else {
		e.up.Set(1)
	}

	e.up.Collect(ch)
	e.dhcpEnabled.Collect(ch)
	e.natEnabled.Collect(ch)
	e.leaseInfo.Collect(ch)
	e.leaseOnline.Collect(ch)
	e.leaseLastSeen.Collect(ch)
}

func (e *Exporter) scrape() error {
	// Reset lease vectors each scrape to drop stale leases.
	e.leaseInfo.Reset()
	e.leaseOnline.Reset()
	e.leaseLastSeen.Reset()

	if running, err := dnsmasqRunning(); err != nil {
		log.Printf("dnsmasq check failed: %v", err)
		e.dhcpEnabled.Set(0)
	} else {
		e.dhcpEnabled.Set(boolToFloat(running))
	}

	if enabled, err := natEnabled(e.wanInterface); err != nil {
		log.Printf("nat check failed: %v", err)
		e.natEnabled.Set(0)
	} else {
		e.natEnabled.Set(boolToFloat(enabled))
	}

	leases, err := readLeases(e.leaseFile)
	if err != nil {
		// Missing lease file is not fatal; just log and continue with zero leases.
		log.Printf("lease read failed: %v", err)
		leases = nil
	}

	now := time.Now()
	for _, l := range leases {
		host := l.host
		if host == "*" || host == "" {
			host = "unknown"
		}
		e.leaseInfo.WithLabelValues(l.mac, host, l.ip).Set(1)

		lastSeen, ok := e.getLastSeen(l.ip)
		var ageSeconds float64
		if ok {
			ageSeconds = now.Sub(lastSeen).Seconds()
		} else {
			ageSeconds = -1
		}

		online := 0.0
		if ok && now.Sub(lastSeen) < e.onlineThreshold {
			online = 1
		}

		e.leaseOnline.WithLabelValues(l.mac, host, l.ip).Set(online)
		e.leaseLastSeen.WithLabelValues(l.mac, host, l.ip).Set(ageSeconds)
	}

	return nil
}

func (e *Exporter) recordSeen(ip string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastSeen[ip] = time.Now()
}

func (e *Exporter) getLastSeen(ip string) (time.Time, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	ts, ok := e.lastSeen[ip]
	return ts, ok
}

func boolToFloat(v bool) float64 {
	if v {
		return 1
	}
	return 0
}
