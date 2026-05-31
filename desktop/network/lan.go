package network

import (
	"errors"
	"net"
)

type LANDetector struct{}

func NewLANDetector() *LANDetector {
	return &LANDetector{}
}

func (d *LANDetector) LocalIPv4() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() || !isPrivateLAN(ip) {
				continue
			}
			return ip.String(), nil
		}
	}

	return "", errors.New("no LAN IPv4 address found")
}

func isPrivateLAN(ip net.IP) bool {
	return ip[0] == 10 ||
		(ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) ||
		(ip[0] == 192 && ip[1] == 168)
}
