package exporter

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func natEnabled(wanInterface string) (bool, error) {
	forwardBytes, err := os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	if err != nil {
		return false, err
	}
	forward := strings.TrimSpace(string(forwardBytes)) == "1"

	masq, err := hasMasqueradeRule(wanInterface)
	if err != nil {
		return false, err
	}

	return forward && masq, nil
}

func hasMasqueradeRule(wanInterface string) (bool, error) {
	cmd := exec.Command("iptables", "-t", "nat", "-S")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(output), "\n")
	needle := fmt.Sprintf("-A POSTROUTING -o %s -j MASQUERADE", wanInterface)
	for _, line := range lines {
		if strings.Contains(line, "MASQUERADE") && strings.Contains(line, "-o "+wanInterface) {
			// Accept variations that include source or comment.
			if strings.HasPrefix(line, "-A POSTROUTING") {
				return true, nil
			}
		}
		// Exact match fast path.
		if line == needle {
			return true, nil
		}
	}
	return false, nil
}
