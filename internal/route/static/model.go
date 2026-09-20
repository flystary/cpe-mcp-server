package static

// Destination string  描述
// Netmask     string  子网掩码
// Nexthop     string  下一跳地址
// Interface   string  下一跳网络接口
// Preference  int
// Metric      int
// Track       bool

// StaticRouteSpec 纯数据模型
type StaticRouteSpec struct {
	Destination string `json:"destination"`
	Netmask     string `json:"netmask"`
	Nexthop     string `json:"nexthop"`
	Interface   string `json:"interface,omitempty"`
	Metric      int    `json:"metric,omitempty"`
	Track       bool   `json:"track,omitempty"`
	Status      string `json:"status,omitempty"`
}
