package netutil

import (
	"fmt"
	"net/netip"
)

// ParseNetmaskToBits 将 "255.255.255.0" 快速转化为 24 位前缀
func ParseNetmaskToBits(maskStr string) (int, error) {
	addr, err := netip.ParseAddr(maskStr)
	if err != nil {
		return 0, fmt.Errorf("invalid netmask format: %w", err)
	}

	if !addr.Is4() {
		return 0, fmt.Errorf("only IPv4 netmask is supported currently")
	}

	bytes := addr.As4()
	ones := 0
	// 逐个字节统计连续的 1 的个数
	for _, b := range bytes {
		for b > 0 {
			if b&1 == 1 {
				ones++
			}
			b >>= 1
		}
	}

	// 校验子网掩码合法性（必须是连续的 1，比如 255.255.128.0 的 ones 是 17）
	if ones < 0 || ones > 32 {
		return 0, fmt.Errorf("netmask bits out of standard range: %d", ones)
	}

	return ones, nil
}

// GetCidrAndNetwork 传入 IP 和掩码，计算并返回标准对齐的 CIDR 和纯网络号
// 平替 eval $(ipcalc -4nmp $ip $netmask)
func GetCidrAndNetwork(ipStr, maskStr string) (cidr string, network string, err error) {
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return "", "", fmt.Errorf("invalid destination ip: %w", err)
	}

	bits, err := ParseNetmaskToBits(maskStr)
	if err != nil {
		return "", "", err
	}

	// 通过 Masked() 强行清空主机位
	prefix := netip.PrefixFrom(ip, bits).Masked()

	return prefix.String(), prefix.Addr().String(), nil
}
