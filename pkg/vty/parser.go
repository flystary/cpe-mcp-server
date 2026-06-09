package vty

import "encoding/json"

// RouteEntry 标准的 FRR 路由表项结构
type RouteEntry struct {
	Protocol string `json:"protocol"`
	Selected bool   `json:"selected"`
	Nexthop  string `json:"nexthop"`
}

// ParseFRRJSON 使用泛型将原始 JSON 解析为指定的结构体 T
func ParseFRRJSON[T any](raw string) (T, error) {
	var result T
	err := json.Unmarshal([]byte(raw), &result)
	return result, err
}
