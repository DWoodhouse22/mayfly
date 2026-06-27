package tunnel

import (
	"fmt"
	"mayfly/internal/config"
	"mayfly/internal/keygen"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

func Up(config *config.ClientConfig) (*device.Device, error) {
	tunDevice, err := tun.CreateTUN("mayfly0", 1420)
	if err != nil {
		return nil, fmt.Errorf("creating TUN: %w", err)
	}

	dev := device.NewDevice(tunDevice, conn.NewDefaultBind(), device.NewLogger(device.LogLevelVerbose, "mayfly "))
	hexPrivateKey, err := keygen.EncodeToHex(config.PrivateKey)
	if err != nil {
		return nil, err
	}
	hexServerPublicKey, err := keygen.EncodeToHex(config.ServerPublicKey)
	if err != nil {
		return nil, err
	}
	uapiConf := fmt.Sprintf("private_key=%s\npublic_key=%s\nendpoint=%s\nallowed_ip=0.0.0.0/0\n",
		hexPrivateKey, hexServerPublicKey, config.Endpoint)
	if err := dev.IpcSet(uapiConf); err != nil {
		return nil, fmt.Errorf("configuring wireguard device: %w", err)
	}

	if err := dev.Up(); err != nil {
		return nil, fmt.Errorf("starting wireguard tunnel: %w", err)
	}

	if err := setupRouting(tunDevice, config); err != nil {
		return nil, err
	}

	return dev, nil
}

func Down(dev *device.Device) error {
	if err := dev.Down(); err != nil {
		return fmt.Errorf("tearing down wireguard tunnel: %w", err)
	}

	if err := teardownRouting(); err != nil {
		return fmt.Errorf("tearing down routing: %w", err)
	}

	return nil
}
