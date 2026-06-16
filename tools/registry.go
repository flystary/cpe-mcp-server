package tools

import (
	"cpe-mcp-server/pkg/mcp"
	"sync"
)

var (
	mu       sync.RWMutex
	toolPool = make(map[string]mcp.Tool)
)

// Register 给各独立原子工具在 init() 中调用
func Register(name string, desc string, schema interface{}, handler mcp.ToolHandler) {
	mu.Lock()
	defer mu.Unlock()
	toolPool[name] = mcp.Tool{
		Def: mcp.ToolDefinition{
			Name:        name,
			Description: desc,
			InputSchema: schema,
		},
		Handler: handler,
	}
}

// Setup 全量将当前目录已加载的独立工具灌入 MCP 注册中心
func Setup(reg *mcp.Registry) {
	mu.RLock()
	defer mu.RUnlock()
	for _, t := range toolPool {
		reg.RegisterTool(t.Def.Name, t.Def.Description, t.Def.InputSchema, t.Handler)
	}
}
