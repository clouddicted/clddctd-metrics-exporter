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
sudo setcap cap_net_raw+ep ./clddctd-metrics-exporter
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
- `-ping-cycle-duration` (default `5s`): duration of each ping cycle; every lease is pinged once at a random instant < this duration, then a summary is logged.
- `-log-pings` (default `false`): also log each successful ping (failures and per-cycle summaries are always logged). Logs are emitted in logfmt.

## OpenRC service (example)
1) Install binary into your PATH (e.g., `/usr/local/bin/clddctd-metrics-exporter`) and set capability:
```sh
sudo install -m 0755 clddctd-metrics-exporter /usr/local/bin/
sudo setcap cap_net_raw+ep /usr/local/bin/clddctd-metrics-exporter
```
2) Copy service files:
```sh
sudo install -m 0644 contrib/openrc/clddctd-metrics-exporter.conf /etc/conf.d/clddctd-metrics-exporter
sudo install -m 0755 contrib/openrc/clddctd-metrics-exporter /etc/init.d/clddctd-metrics-exporter
```
3) Enable and start:
```sh
sudo rc-update add clddctd-metrics-exporter default
sudo rc-service clddctd-metrics-exporter start
```
The conf.d file is empty by default; set `EXPORTER_OPTS` there if you want to override defaults.
