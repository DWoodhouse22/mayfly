package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func startUnbound() (*exec.Cmd, error) {
	cmd := exec.Command("unbound", "-d", "-c", "/etc/unbound/custom.conf")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting unbound: %w", err)
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("unbound exited: %v", err)
		}
	}()
	return cmd, nil
}
