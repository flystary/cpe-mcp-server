package mcp

import (
	"encoding/json"
	"fmt"
	"log"
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

func (r *Registry) RegisterService(
	name string,
	enable bool,
	factory ServiceFactory,
) {
	if !enable {
		return
	}

	svc := factory()
	key := strings.ToLower(name)

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

			var req struct {
				Action string `json:"action"`
			}

			if err := json.Unmarshal(args, &req); err != nil {
				return nil, err
			}

			return svc.Execute(
				ctx.Context,
				req.Action,
				args,
			)
		},
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
func (r *Registry) ExecuteTool(
	ctx ToolContext,
	name string,
	args json.RawMessage,
) (any, error) {

	r.mu.RLock()
	tool, ok := r.tools[strings.ToLower(name)]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	return tool.Handler(ctx, args)
}

func (r *Registry) RegisterTool(def ToolDefinition, handler ToolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools[strings.ToLower(def.Name)] = &Tool{
		Def:     def,
		Handler: handler,
	}
}

func (r *Registry) Dump() {

	r.mu.RLock()
	defer r.mu.RUnlock()

	log.Printf(
		"[MCP] services=%d tools=%d",
		len(r.services),
		len(r.tools),
	)
	for name := range r.services {
		log.Printf(
			"  service: %s",
			name,
		)
	}
	for name := range r.tools {
		log.Printf(
			"  tool: %s",
			name,
		)
	}
}
