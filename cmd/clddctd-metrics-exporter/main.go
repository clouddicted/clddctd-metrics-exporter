package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	exporter "clddctd-metrics-exporter/internal/exporter"
)

const (
	defaultLeaseFile    = "/var/lib/misc/dnsmasq.leases"
	defaultListenAddr   = ":9100"
	defaultPingWorkers  = 5
	defaultOnlineWindow = 60 * time.Second
	defaultPingTimeout  = 500 * time.Millisecond
	defaultWanInterface = "wan0"
)

func main() {
	listenAddr := flag.String("listen", defaultListenAddr, "HTTP listen address")
	leaseFile := flag.String("lease-file", defaultLeaseFile, "dnsmasq lease file path")
	wanInterface := flag.String("wan-interface", defaultWanInterface, "WAN interface name for NAT check")
	onlineWindow := flag.Duration("online-window", defaultOnlineWindow, "Duration a host stays online after successful ping")
	pingTimeout := flag.Duration("ping-timeout", defaultPingTimeout, "Per-host ping timeout")
	pingWorkers := flag.Int("ping-workers", defaultPingWorkers, "Concurrent ping workers")
	flag.Parse()

	exp := exporter.NewExporter(*leaseFile, *wanInterface, *onlineWindow, *pingTimeout, *pingWorkers)

	registry := prometheus.NewRegistry()
	registry.MustRegister(exp)
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(prometheus.NewGoCollector())

	httpHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	http.Handle("/metrics", httpHandler)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("starting gateway exporter on %s", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
