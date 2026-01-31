package exporter

import "os/exec"

func dnsmasqRunning() (bool, error) {
	cmd := exec.Command("pidof", "dnsmasq")
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				// Not running.
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}
