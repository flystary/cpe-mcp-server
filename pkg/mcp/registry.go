package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

type ServiceFactory func() Service

type Registry struct {
	mu sync.RWMutex

	services map[string]Service
	tools    map[string]*Tool
}

func NewRegistry() *Registry {
	return &Registry{
		services: make(map[string]Service),
		tools:    make(map[string]*Tool),
	}
}

func (r *Registry) RegisterService(name string, enable bool, factory ServiceFactory) {
	if !enable {
		return
	}

	svc := factory()
	key := strings.ToLower(name)

	rawSchema := svc.Schema()
	fmt.Printf("%#v\n", rawSchema)

	mcpInputSchema := map[string]any{
		"type":        "object",
		"description": "动态网络子模块编排总线入口",
		"properties":  make(map[string]any),
		"required":    []string{"protocol", "action"},
	}

	if props, ok := rawSchema["properties"]; ok {
		if jsonBytes, err := json.Marshal(props); err == nil {
			var normalizedProps map[string]any
			if json.Unmarshal(jsonBytes, &normalizedProps) == nil {
				mcpInputSchema["properties"] = normalizedProps
			}
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.services[key] = svc
	r.tools[key] = &Tool{
		Def: ToolDefinition{
			Name:        key,
			Description: fmt.Sprintf("%s service", key),
			InputSchema: svc.Schema(),
		},
		Handler: func(ctx ToolContext, args json.RawMessage) (any, error) {

			var actionReq struct {
				Action string `json:"action"`
			}

			if err := json.Unmarshal(args, &actionReq); err != nil {
				return nil, err
			}

			return svc.Execute(ctx.Context, actionReq.Action, args)
		},
	}
}

// RegisterTool 注册原子化的独立轻量级 MCP 工具
func (r *Registry) RegisterTool(name string, desc string, schema interface{}, handler ToolHandler) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools[key] = &Tool{
		Def: ToolDefinition{
			Name:        key,
			Description: desc,
			InputSchema: schema,
		},
		Handler: handler,
	}
}

func (r *Registry) ToolList() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t.Def)
	}
	return out
}

func (r *Registry) ExecuteTool(ctx ToolContext, name string, args json.RawMessage) (any, error) {
	r.mu.RLock()
	tool, ok := r.tools[strings.ToLower(name)]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}
	return tool.Handler(ctx, args)
}

func (r *Registry) Dump() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fmt.Println(strings.Repeat("═", 85))
	fmt.Printf(" 🚀 [CPE-MCP-SERVER] 核心网关控制平面初始化快照 (已注册 Service & Tools 清单)\n")
	fmt.Println(strings.Repeat("═", 85))

	for name, tool := range r.tools {
		_, isService := r.services[name]

		if isService {
			// 服务容器模式：深度扫描解算内部 Actions 与子驱动参数
			fmt.Printf(" 🏢 [常驻服务型容器] Tool Name: %-15s | 描述: %s\n", fmt.Sprintf("[%s]", tool.Def.Name), tool.Def.Description)
			// 漂亮打印微内核底层的动态路由 Schema 关系
			if schemaMap, ok := tool.Def.InputSchema.(map[string]any); ok {
				if props, ok := schemaMap["properties"].(map[string]any); ok {
					for protoName, protoSchema := range props {
						fmt.Printf("    ├── 📡 协议子模块 (Protocol): %s\n", protoName)

						if protoMap, ok := protoSchema.(map[string]any); ok {
							if actions, ok := protoMap["Actions"].(map[string]any); ok {

								for actName, actDetail := range actions {
									fmt.Printf("    │    ├── ⚡ 动作 (Action): %s\n", actName)
									if detail, ok := actDetail.(map[string]any); ok {
										if fields, ok := detail["Fields"].([]any); ok {

											for _, f := range fields {
												if fieldMap, ok := f.(map[string]any); ok {
													reqStr := "可选"
													if req, ok := fieldMap["Required"].(bool); ok && req {
														reqStr = "必填"
													}
													fmt.Printf("    │    │    └── 📝 参数: %-12s | 类型: %-6s | %s\n",
														fieldMap["Name"], fieldMap["Type"], reqStr)
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		} else {
			// tools 目录下的独立原子工具模式
			fmt.Printf(" 🧪 [独立原子型插件] Tool Name: %-15s | 描述: %s\n", fmt.Sprintf("[%s]", tool.Def.Name), tool.Def.Description)
			if schemaBytes, err := json.Marshal(tool.Def.InputSchema); err == nil && string(schemaBytes) != "null" {
				fmt.Printf("    └── 📋 参数契约 Schema: %s\n", string(schemaBytes))
			}
		}
		fmt.Println(strings.Repeat("─", 85))
	}
}
