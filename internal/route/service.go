package route

import (
	"context"
	"cpe-mcp-server/pkg/mcp"
	"encoding/json"
	"fmt"
)

type Service struct {
	engine *Engine
}

func NewService(e *Engine) *Service {
	return &Service{
		engine: e,
	}
}

func (s *Service) New() mcp.Service {
	return NewService(s.engine)
}

func (s *Service) Name() string {
	return "route"
}

func (s *Service) Schema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": s.engine.Schema(),
	}
}

func (s *Service) Execute(ctx context.Context, action string, raw json.RawMessage) (interface{}, error) {

	// MCP 标准结构
	var req struct {
		Protocol string `json:"protocol"`
	}

	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if req.Protocol == "" {
		return nil, fmt.Errorf("missing protocol")
	}

	// 调用 route engine
	err := s.engine.Dispatch(ctx, req.Protocol, action, raw)
	if err != nil {
		return map[string]any{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	return map[string]any{
		"success":  true,
		"protocol": req.Protocol,
		"action":   action,
	}, nil
}

func (s *Service) ReadResource(ctx context.Context, protocol string) (interface{}, error) {
	return s.engine.List(ctx, protocol)
}

func (s *Service) Actions() []string {
	actionSet := make(map[string]struct{})
	modules := s.engine.GetModulesSnapshot()
	for _, mod := range modules {
		for _, act := range mod.Actions() {
			actionSet[act] = struct{}{}
		}
	}

	var list []string
	for act := range actionSet {
		list = append(list, act)
	}
	return list
}

func (s *Service) DebugDump() {
	for name := range s.engine.modules {
		fmt.Printf("[route] loaded module: %s\n", name)
	}
}
