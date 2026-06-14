package static

// StaticRouteSpec 纯数据模型（无逻辑）
type StaticRouteSpec struct {
	Destination string `json:"destination"`
	Netmask     string `json:"netmask"`
	Nexthop     string `json:"nexthop"`
	Interface   string `json:"interface,omitempty"`
	Metric      int    `json:"metric,omitempty"`
	Track       bool   `json:"track,omitempty"`
}
