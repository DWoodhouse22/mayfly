//go:build windows

package tunnel

import (
	"fmt"
	"mayfly/internal/config"
	"net"
	"net/netip"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

var (
	originalLUID    winipcfg.LUID
	originalGateway netip.Addr
	tunnelLUID      winipcfg.LUID
	vpsAddr         netip.Addr
)

func setupRouting(tunDevice tun.Device, config *config.ClientConfig) error {
	nativeTun := tunDevice.(*tun.NativeTun)
	tunnelLUID = winipcfg.LUID(nativeTun.LUID())
	if err := assignAddress(tunnelLUID, config.ClientIP); err != nil {
		return err
	}

	gateway, wanLUID, err := getDefaultRoute()
	if err != nil {
		return fmt.Errorf("finding default gateway: %w", err)
	}
	originalGateway = gateway
	originalLUID = wanLUID
	// clean up any leftover route from a previous unclean shutdown
	host, _, err := net.SplitHostPort(config.Endpoint)
	if err != nil {
		return err
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return err
	}
	vpsAddr = addr

	originalLUID.DeleteRoute(netip.PrefixFrom(vpsAddr, 32), originalGateway)
	if err := originalLUID.AddRoute(netip.PrefixFrom(vpsAddr, 32), originalGateway, 0); err != nil {
		return fmt.Errorf("adding VPS host route: %w", err)
	}

	tunnelGateway, err := netip.ParseAddr(config.DNS)
	if err != nil {
		return err
	}

	if err := tunnelLUID.AddRoute(netip.MustParsePrefix("0.0.0.0/0"), tunnelGateway, 0); err != nil {
		return fmt.Errorf("adding default route: %w", err)
	}

	return tunnelLUID.SetDNS(windows.AF_INET, []netip.Addr{tunnelGateway}, nil)
}

func getDefaultRoute() (netip.Addr, winipcfg.LUID, error) {
	table, err := winipcfg.GetIPForwardTable2(windows.AF_INET)
	if err != nil {
		return netip.Addr{}, 0, err
	}
	var best *winipcfg.MibIPforwardRow2
	for i := range table {
		row := &table[i]
		if row.DestinationPrefix.Prefix().Bits() == 0 {
			if best == nil || row.Metric < best.Metric {
				best = row
			}
		}
	}
	if best == nil {
		return netip.Addr{}, 0, fmt.Errorf("no default gateway found")
	}
	return best.NextHop.Addr(), best.InterfaceLUID, nil
}

func assignAddress(luid winipcfg.LUID, clientIP string) error {
	addr, err := netip.ParseAddr(clientIP)
	if err != nil {
		return err
	}
	return luid.AddIPAddress(netip.PrefixFrom(addr, 32))
}

func teardownRouting() error {
	if err := originalLUID.DeleteRoute(netip.PrefixFrom(vpsAddr, 32), originalGateway); err != nil {
		return err
	}
	return tunnelLUID.SetDNS(windows.AF_INET, nil, nil)
}
