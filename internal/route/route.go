package route

import (
	"context"
	"cpe-mcp-server/pkg/mcp"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

type RouteHandler interface {
	Add(ctx context.Context, args json.RawMessage) (interface{}, error)
	Set(ctx context.Context, args json.RawMessage) (interface{}, error)
	Del(ctx context.Context, args json.RawMessage) (interface{}, error)
	List(ctx context.Context) (interface{}, error)
}

type AssetFactory func() RouteHandler

type RouteController struct {
	mu     sync.RWMutex
	assets map[string]RouteHandler
}

var (
	registryMu    sync.Mutex
	assetRegistry = make(map[string]AssetFactory)
)

// RegisterRoutingAsset 供各平行路由资产（如 static）在 init() 中自发现注册
func RegisterRoutingAsset(name string, enabled bool, factory AssetFactory) {
	if !enabled {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	assetRegistry[strings.ToLower(name)] = factory
}

func NewRouteService() mcp.Service {
	registryMu.Lock()
	defer registryMu.Unlock()
	ctrl := &RouteController{assets: make(map[string]RouteHandler)}
	for name, factory := range assetRegistry {
		ctrl.assets[name] = factory()
	}
	return ctrl
}

func init() { mcp.RegisterService("route", true, NewRouteService) }

func (c *RouteController) Name() string { return "route" }

// GetSchema 动态向北向暴露当前支持的路由大类（如 static、bgp）
func (c *RouteController) GetSchema() interface{} {
	c.mu.RLock()
	var list []string
	for k := range c.assets {
		list = append(list, k)
	}
	c.mu.RUnlock()

	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"routing_type": map[string]interface{}{"type": "string", "enum": list},
			"action":       map[string]interface{}{"type": "string", "enum": []string{"add", "set", "del", "list"}},
			"arguments":    map[string]string{"type": "object"},
		},
		"required": []string{"routing_type", "action"},
	}
}

func (c *RouteController) ReadResource(ctx context.Context, p string) (interface{}, error) {
	return nil, nil
}

// ExecuteTool 北向 MCP 总线切入点，根据 action 动词精准降维调用
func (c *RouteController) ExecuteTool(ctx context.Context, action string, rawArgs json.RawMessage) (interface{}, error) {
	var baseArgs struct {
		RoutingType string `json:"routing_type"`
	}
	if err := json.Unmarshal(rawArgs, &baseArgs); err != nil {
		return nil, err
	}

	c.mu.RLock()
	asset, exists := c.assets[strings.ToLower(baseArgs.RoutingType)]
	c.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("routing asset '%s' offline or uninitialized", baseArgs.RoutingType)
	}

	// 根据标准动词进行强类型分发
	switch strings.ToLower(action) {
	case "add":
		return asset.Add(ctx, rawArgs)
	case "set":
		return asset.Set(ctx, rawArgs)
	case "del":
		return asset.Del(ctx, rawArgs)
	case "list":
		return asset.List(ctx)
	default:
		return nil, fmt.Errorf("unsupported routing action: %s", action)
	}
}
