package internal

import "net"

func IsValidInterface(iface string) bool {
	if len(iface) == 0 || len(iface) > 16 {
		return false
	}
	for _, ch := range iface {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '.') {
			return false
		}
	}
	return true
}

func IsValidIP(ipStr string) bool { return net.ParseIP(ipStr) != nil }
