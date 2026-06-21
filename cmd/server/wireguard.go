package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	tunnelAddr = "10.0.0.1/24"
	subnet     = "10.0.0.0/24"
	ifacename  = "wg0"
	listenPort = 51820
)

type wgServer struct {
	mu        sync.Mutex
	dev       *device.Device
	PublicKey wgtypes.Key
}

func newWGServer() (*wgServer, error) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	tunDev, err := tun.CreateTUN(ifacename, device.DefaultMTU)
	if err != nil {
		return nil, err
	}

	logger := device.NewLogger(device.LogLevelError, "")
	dev := device.NewDevice(tunDev, conn.NewDefaultBind(), logger)

	if err := dev.IpcSet(fmt.Sprintf(
		"private_key=%s\nlisten_port=%d\n",
		hex.EncodeToString(privateKey[:]),
		listenPort,
	)); err != nil {
		dev.Close()
		return nil, err
	}

	if err := dev.Up(); err != nil {
		dev.Close()
		return nil, err
	}

	if err := configureInterface(); err != nil {
		dev.Close()
		return nil, err
	}

	return &wgServer{dev: dev, PublicKey: privateKey.PublicKey()}, nil
}

func configureInterface() error {
	if err := runCmd("ip", "addr", "add", tunnelAddr, "dev", ifacename); err != nil {
		return err
	}

	if err := runCmd("ip", "link", "set", ifacename, "up"); err != nil {
		return err
	}

	if err := runCmd("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", subnet, "!", "-d", subnet, "-j", "MASQUERADE"); err != nil {
		return err
	}

	return nil
}

func runCmd(args ...string) error {
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *wgServer) AddPeer(pubKey wgtypes.Key, ip net.IP) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.dev.IpcSet(fmt.Sprintf(
		"public_key=%s\nallowed_ip=%s/32\npersistent_keepalive_interval=25\n",
		hex.EncodeToString(pubKey[:]),
		ip.String(),
	))
}

func (s *wgServer) Close() {
	s.dev.Close()
}
