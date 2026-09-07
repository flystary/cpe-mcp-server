package static

// Destination string
// Netmask     string
// Nexthop     string
// Interface   string
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
}
