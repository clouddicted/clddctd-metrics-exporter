package exporter

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

type lease struct {
	expiry time.Time
	mac    string
	ip     string
	host   string
}

func readLeases(path string) ([]lease, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	leases := make([]lease, 0)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		expiryUnix, err := parseExpiry(fields[0])
		if err != nil {
			continue
		}
		leases = append(leases, lease{
			expiry: expiryUnix,
			mac:    fields[1],
			ip:     fields[2],
			host:   fields[3],
		})
	}

	if err := scanner.Err(); err != nil {
		return leases, err
	}

	return leases, nil
}

func parseExpiry(raw string) (time.Time, error) {
	if raw == "0" {
		// Infinite lease, but we still return now.
		return time.Now(), nil
	}
	sec, err := parseInt(raw)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(sec, 0), nil
}

func parseInt(s string) (int64, error) {
	var n int64
	_, err := fmt.Sscan(s, &n)
	if err != nil {
		return 0, err
	}
	return n, nil
}
