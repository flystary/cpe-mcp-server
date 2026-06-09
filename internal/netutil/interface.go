package netutil

import (
	"fmt"
	"net"
	"strings"
)

// IsLogicalInterface 判定是否是 CPE 体系定义的逻辑网卡别名
func IsLogicalInterface(s string) bool {
	u := strings.ToUpper(strings.TrimSpace(s))
	return strings.HasPrefix(u, "WAN") || strings.HasPrefix(u, "LAN") ||
		strings.HasPrefix(u, "ETH") || strings.HasPrefix(u, "MOBILE") ||
		strings.HasPrefix(u, "VLAN") || strings.HasPrefix(u, "BOND")
}

// FindRealInterface 纽带映射：读取环境上下文，将大写别名翻译为物理网卡名
func FindRealInterface(logicalName string) (string, error) {
	target := strings.ToUpper(strings.TrimSpace(logicalName))
	if target == "" {
		return "", fmt.Errorf("interface name cannot be empty")
	}

	if target == "WAN1" {
		// 校验 Linux 宿主机物理设备是否存在
		if _, err := net.InterfaceByName("ppp0"); err != nil {
			return "", fmt.Errorf("mapped physical device ppp0 is down or missing: %w", err)
		}
		return "ppp0", nil
	}

	// 默认兜底：如果没有映射关系，转为小写直接尝试对齐物理网卡
	lowerName := strings.ToLower(logicalName)
	if _, err := net.InterfaceByName(lowerName); err != nil {
		return "", fmt.Errorf("physical interface %s does not exist in linux kernel: %w", lowerName, err)
	}

	return lowerName, nil
}

// InterfaceToPppoe 动态提取 pppoe 拨号口的虚拟下一跳网关 IP
func InterfaceToPppoe(iface string) (string, error) {
	ief, err := net.InterfaceByName(iface)
	if err != nil {
		return "", fmt.Errorf("failed to look up interface %s: %w", iface, err)
	}

	addrs, err := ief.Addrs()
	if err != nil {
		return "", err
	}

	// 遍历绑定的物理/虚拟 IP 列表，提取干净的 IPv4 地址作为网关下一跳
	for _, addr := range addrs {
		ipStr := addr.String()
		// ppp 拨号口通常返回一个 32 位的点对点掩码地址（例如 100.64.1.2/32）
		if !strings.Contains(ipStr, ":") { // 过滤掉 IPv6
			pureIP := strings.Split(ipStr, "/")[0]
			return pureIP, nil
		}
	}

	return "", fmt.Errorf("no valid ipv4 gateway found on pppoe interface %s", iface)
}
