//go:build linux

package tunnel

import (
	"fmt"
	"mayfly/internal/config"
	"net"
	"os"
	"os/exec"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/tun"
)

var (
	origVpsIp           net.IP
	origGw              net.IP
	origLIdx            int
	tunLIdx             int
	origResolvConf      []byte
	usedSystemdResolved bool
)

func setupRouting(tunDevice tun.Device, config *config.ClientConfig) error {
	link, err := netlink.LinkByName("mayfly0")
	if err != nil {
		return fmt.Errorf("getting net link: %w", err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("setting up net link: %w", err)
	}

	addr := &netlink.Addr{IPNet: &net.IPNet{
		IP:   net.ParseIP(config.ClientIP),
		Mask: net.CIDRMask(32, 32),
	}}
	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("assigning IP to mayfly0: %w", err)
	}

	routes, err := netlink.RouteList(nil, 0)
	if err != nil {
		return fmt.Errorf("listing routes: %w", err)
	}
	var best *netlink.Route
	for i := range routes {
		r := &routes[i]
		if r.Dst == nil || r.Dst.String() == "0.0.0.0/0" {
			if best == nil || r.Priority < best.Priority {
				best = r
			}
		}
	}
	if best == nil {
		return fmt.Errorf("no default route found")
	}
	origGw = best.Gw
	origLIdx = best.LinkIndex

	host, _, _ := net.SplitHostPort(config.Endpoint)
	origVpsIp = net.ParseIP(host)
	if err := netlink.RouteReplace(&netlink.Route{
		LinkIndex: origLIdx,
		Gw:        origGw,
		Dst:       &net.IPNet{IP: origVpsIp, Mask: net.CIDRMask(32, 32)},
	}); err != nil {
		return fmt.Errorf("adding VPS host route: %w", err)
	}

	tunLIdx = link.Attrs().Index
	if err := netlink.RouteReplace(&netlink.Route{
		LinkIndex: tunLIdx,
		Dst:       &net.IPNet{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)},
	}); err != nil {
		return fmt.Errorf("adding default tunnel route: %w", err)
	}

	usedSystemdResolved = isSystemdResolved()
	if usedSystemdResolved {
		if err := exec.Command("resolvectl", "dns", "mayfly0", config.DNS).Run(); err != nil {
			return fmt.Errorf("setting DNS: %w", err)
		}
		if err := exec.Command("resolvectl", "domain", "mayfly0", "~.").Run(); err != nil {
			restoreDNS() // atempt cleanup
			return fmt.Errorf("setting DNS domain: %w", err)
		}
	} else {
		// fallback to /etc/resolv.conf
		const resolvPath = "/etc/resolv.conf"
		original, err := os.ReadFile(resolvPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", resolvPath, err)
		}
		origResolvConf = original
		info, err := os.Stat(resolvPath)
		if err != nil {
			return fmt.Errorf("stating %s: %w", resolvPath, err)
		}
		if err := writeResolvConf([]byte("nameserver "+config.DNS+"\n"), info.Mode()); err != nil {
			return fmt.Errorf("writing %s: %w", resolvPath, err)
		}
	}

	return nil
}

func isSystemdResolved() bool {
	return exec.Command("systemctl", "is-active", "--quiet", "systemd-resolved").Run() == nil
}

func teardownRouting() error {
	if err := netlink.RouteDel(&netlink.Route{
		LinkIndex: origLIdx,
		Dst:       &net.IPNet{IP: origVpsIp, Mask: net.CIDRMask(32, 32)},
		Gw:        origGw,
	}); err != nil {
		return fmt.Errorf("removing VPS host route: %w", err)
	}
	if err := netlink.RouteDel(&netlink.Route{
		LinkIndex: tunLIdx,
		Dst:       &net.IPNet{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)},
	}); err != nil {
		return fmt.Errorf("removing tunnel default route: %w", err)
	}
	return restoreDNS()
}

func restoreDNS() error {
	if usedSystemdResolved {
		if err := exec.Command("resolvectl", "revert", "mayfly0").Run(); err != nil {
			return fmt.Errorf("reverting DNS via resolvectl: %w", err)
		}
		return nil
	}
	if err := writeResolvConf(origResolvConf, 0644); err != nil {
		return fmt.Errorf("restoring /etc/resolv.conf: %w", err)
	}
	return nil
}

func writeResolvConf(content []byte, mode os.FileMode) error {
	const resolvPath = "/etc/resolv.conf"
	tmpFile, err := os.CreateTemp("/etc", "resolv.conf.*")
	if err != nil {
		return err
	}
	if _, err := tmpFile.Write(content); err != nil {
		os.Remove(tmpFile.Name())
		return err
	}
	tmpFile.Close()
	if err := os.Chmod(tmpFile.Name(), mode); err != nil {
		os.Remove(tmpFile.Name())
		return err
	}
	if err := os.Rename(tmpFile.Name(), resolvPath); err != nil {
		os.Remove(tmpFile.Name())
		return err
	}
	return nil
}
