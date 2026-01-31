# Gateway Prometheus Exporter

Exports DHCP/NAT status and per-lease reachability from the Alpine-based gateway.

## Metrics
- `gateway_up` — exporter scrape success.
- `gateway_dhcp_enabled` — 1 if `dnsmasq` process is running.
- `gateway_nat_enabled` — 1 if `net.ipv4.ip_forward` is `1` **and** an iptables `MASQUERADE` rule for `wan0` exists.
- `gateway_dhcp_lease_info{mac,host,ip}` — static gauge 1 per lease.
- `gateway_dhcp_lease_online{mac,host,ip}` — 1 if last successful ping < 60s (pings run in background).
- `gateway_dhcp_lease_last_seen_seconds{mac,host,ip}` — seconds since last successful ping (`-1` if never).

## Defaults
- Lease file: `/var/lib/misc/dnsmasq.leases` (single pool, no static reservations).
- Interfaces: LAN `eth0`, WAN `wan0`.
- Ping window: 60s; timeout: 500ms; workers: 5.
- Listen: `:9100`.

## Build
```sh
GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod go build -o clddctd-metrics-exporter ./cmd/clddctd-metrics-exporter
```

## Run (example)
```sh
sudo setcap cap_net_raw,cap_net_admin+ep ./clddctd-metrics-exporter
./clddctd-metrics-exporter \
  -lease-file /var/lib/misc/dnsmasq.leases \
  -wan-interface wan0 \
  -listen :9100
```

Expose `/metrics` for Prometheus and `/healthz` for liveness.

## Flags
- `-listen` (default `:9100`): HTTP listen address.
- `-lease-file` (default `/var/lib/misc/dnsmasq.leases`): path to dnsmasq lease file.
- `-wan-interface` (default `wan0`): interface to check for MASQUERADE rule.
- `-online-window` (default `60s`): time a host stays “online” after a successful ping.
- `-ping-timeout` (default `500ms`): per-host ping timeout.
- `-ping-workers` (default `5`): maximum concurrent ping probes.
- `-ping-cycle-duration-max` (default `5s`): upper bound for each ping cycle; every lease is pinged once at a random instant < this duration, then a summary is logged.
- `-log-pings` (default `false`): also log each successful ping (failures and per-cycle summaries are always logged).
