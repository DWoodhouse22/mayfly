//go:build darwin

package tunnel

import (
	"mayfly/internal/config"

	"golang.zx2c4.com/wireguard/tun"
)

func setupRouting(tunDevice tun.Device, config *config.ClientConfig) error { return nil }

func teardownRouting() error { return nil }
