//go:build linux

package tunnel

import (
	"fmt"
	"mayfly/internal/config"
	"net"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/tun"
)

var (
	origVpsIp net.IP
	origGw    net.IP
	origLIdx  int
)

func setupRouting(tunDevice tun.Device, config *config.ClientConfig) error {
	link, err := netlink.LinkByName("mayfly0")
	if err != nil {
		return err
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return err
	}

	addr := &netlink.Addr{IPNet: &net.IPNet{
		IP:   net.ParseIP(config.ClientIP),
		Mask: net.CIDRMask(32, 32),
	}}
	if err := netlink.AddrAdd(link, addr); err != nil {
		return err
	}

	routes, err := netlink.RouteList(nil, 0)
	if err != nil {
		return err
	}
	var best *netlink.Route
	for i := range routes {
		r := &routes[i]
		if r.Dst == nil {
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
	if err := netlink.RouteAdd(&netlink.Route{
		LinkIndex: origLIdx,
		Gw:        origGw,
		Dst:       &net.IPNet{IP: origVpsIp, Mask: net.CIDRMask(32, 32)},
	}); err != nil {
		return err
	}

	tunnelGw := net.ParseIP(config.DNS)
	if err := netlink.RouteAdd(&netlink.Route{
		LinkIndex: link.Attrs().Index,
		Gw:        tunnelGw,
		Dst:       &net.IPNet{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)},
	}); err != nil {
		return err
	}

	return nil
}

func teardownRouting() error {
	// if err := netlink.RouteDel(&netlink.Route{
	// 	LinkIndex: origLIdx,
	// 	Dst:       &net.IPNet{IP: origVpsIP, Mask: net.CIDRMask(32, 32)},
	// 	Gw:        origGw,
	// }); err != nil {
	// 	return fmt.Errorf("removing VPS host route: %w", err)
	// }
	// return restoreDNS()
	return nil
}
