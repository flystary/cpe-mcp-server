package static

import (
	"context"
	i "cpe-mcp-server/internal"
	"cpe-mcp-server/internal/netutil"
	route "cpe-mcp-server/internal/route"
	"cpe-mcp-server/pkg/vty"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const cliConfDir = "/etc/svxnetworks/routes/static"

var defaultEnable = true

var _ route.RouteHandler = (*Static)(nil)
var _ i.Rollable = (*Static)(nil)

type Static struct{}

// Destination string
// Netmask     string
// Nexthop     string
// Interface   string
// Preference  int
// Metric      int
// Track       bool

func NewStatic() route.RouteHandler { return &Static{} }

func (p *Static) Name() string { return "static" }

func (p *Static) Actions() []string {
	return []string{
		"apply",
		"remove",
		"show",
	}
}

func (s *Static) Validate(action string, args []byte) (map[string]any, error) {

	var spec StaticRouteSpec
	if err := json.Unmarshal(args, &spec); err != nil {
		return nil, err
	}

	switch action {

	case "apply":
		if spec.Destination == "" {
			return nil, fmt.Errorf("missing destination")
		}
		if spec.Netmask == "" {
			return nil, fmt.Errorf("missing netmask")
		}
		if spec.Nexthop == "" {
			return nil, fmt.Errorf("missing nexthop")
		}

	case "remove":
		if spec.Destination == "" {
			return nil, fmt.Errorf("missing destination")
		}
		if spec.Netmask == "" {
			return nil, fmt.Errorf("missing netmask")
		}

	case "show":
		return map[string]any{}, nil

	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}

	// struct → map（统一流转）
	b, _ := json.Marshal(spec)

	var m map[string]any
	_ = json.Unmarshal(b, &m)

	return m, nil
}

func (s *Static) Dispatch(ctx context.Context, action string, args []byte) error {
	var spec StaticRouteSpec
	if err := json.Unmarshal(args, &spec); err != nil {
		return err
	}

	switch action {

	case "apply":
		s.apply(ctx, spec)
	case "remove":
		s.remove(ctx, spec)
	case "show":
		s.show(ctx, spec)

	default:
		return fmt.Errorf("unknown action")
	}

	return nil
}

// apply 执行静态路由下刷
func (s *Static) apply(ctx context.Context, spec StaticRouteSpec) error {
	cidr, network, err := netutil.GetCidrAndNetwork(spec.Destination, spec.Netmask)
	if err != nil {
		return err
	}

	// 唯一性冲突校验
	fileName := fmt.Sprintf("%s.%s", network, spec.Netmask)
	filePath := filepath.Join(cliConfDir, fileName)
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("route entry %s already exists", fileName)
	}

	// 处理下一跳逻辑（黑洞路由或逻辑接口映射）
	target := strings.TrimSpace(spec.Nexthop)
	var finalNexthop string

	if strings.ToLower(target) == "null0" {
		spec.Track = false
		finalNexthop = "null0"
	} else if netutil.IsLogicalInterface(target) {
		spec.Track = false
		realIface, err := netutil.FindRealInterface(target)
		if err != nil {
			return err
		}
		spec.Interface = realIface
		finalNexthop = realIface
	} else {
		finalNexthop = target
	}

	// 特殊网络接口（PPPoE）处理
	if spec.Interface != "" && strings.HasPrefix(spec.Interface, "ppp") {
		if ip, err := netutil.InterfaceToPppoe(spec.Interface); err == nil && ip != "" {
			finalNexthop = ip
		}
	}
	// 路由配置的幂等性
	_ = s.remove(ctx, spec)

	vtyCmds := []string{
		"configure terminal",
		fmt.Sprintf("ip route %s %s", cidr, finalNexthop),
		"end",
		"write",
	}
	if err := vty.ZebraExecute(ctx, vtyCmds); err != nil {
		return err
	}

	// 6. 持久化落盘与状态同步
	_ = os.MkdirAll(cliConfDir, 0755)
	content := fmt.Sprintf("%s %s %t %s\n", cidr, finalNexthop, spec.Track, spec.Interface)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return err
	}

	return nil
}

// remove 执行路由撤销
func (s *Static) remove(ctx context.Context, spec StaticRouteSpec) error {
	_, network, err := netutil.GetCidrAndNetwork(spec.Destination, spec.Netmask)
	if err != nil {
		return err
	}

	var targetNexthop string
	if spec.Interface != "" {
		// 优先使用接口名撤销 (ip route 10.0.0.0/24 eth0)
		targetNexthop = spec.Interface
	} else {
		// 使用 IP 地址撤销
		targetNexthop = spec.Nexthop
	}

	netmask, _ := netutil.ParseNetmaskToBits(spec.Netmask)
	vtyCmds := []string{
		"configure terminal",
		fmt.Sprintf("no ip route %s/%d %s", spec.Destination, netmask, targetNexthop),
		"end",
		"write",
	}

	if err := vty.ZebraExecute(ctx, vtyCmds); err != nil {
		return fmt.Errorf("vty remove route failed: %w", err)
	}

	return os.Remove(filepath.Join(cliConfDir, fmt.Sprintf("%s.%s", network, spec.Netmask)))
}

func (s *Static) show(ctx context.Context, spec StaticRouteSpec) error {
	fmt.Println("show static routes")
	return nil
}

// List 返回当前路由运行数据
func (p *Static) List(ctx context.Context) (interface{}, error) {
	routes, err := vty.ZebraQuery(ctx, "show ip route json")
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(cliConfDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var results []map[string]interface{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		content, _ := os.ReadFile(filepath.Join(cliConfDir, file.Name()))
		parts := strings.Fields(string(content))
		if len(parts) < 4 {
			continue
		}

		cidr, nexthop, track, iface := parts[0], parts[1], parts[2], parts[3]

		status := "inactive"
		if entries, ok := routes[cidr]; ok {
			for _, e := range entries {
				if e.Protocol == "static" && e.Selected {
					status = "active"
					break
				}
			}
		}

		results = append(results, map[string]interface{}{
			"cidr":      cidr,
			"nexthop":   nexthop,
			"track":     track,
			"interface": iface,
			"status":    status,
		})
	}

	return results, nil
}

// Save 执行物理存储同步
func (s *Static) Save(target string) error {
	return syscall.Sync()
}

// Roll 实现配置回滚机制
func (s *Static) Roll(target string) error {
	var spec StaticRouteSpec
	return s.remove(context.Background(), spec)
}

func (s *Static) Schema() route.Schema {
	return route.Schema{
		Actions: map[string]route.ActionSchema{
			"apply": {
				Action: "apply",
				Fields: []route.Field{
					{Name: "destination", Type: route.String, Required: true},
					{Name: "netmask", Type: route.String, Required: true},
					{Name: "nexthop", Type: route.String, Required: true},
					{Name: "interface", Type: route.String},
				},
			},
			"remove": {
				Action: "remove",
				Fields: []route.Field{
					{Name: "destination", Type: route.String, Required: true},
					{Name: "netmask", Type: route.String, Required: true},
				},
			},
		},
	}
}
