package route

import (
	"context"
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

func (s *Service) Name() string {
	return "route"
}

func (s *Service) ExecuteTool(
	ctx context.Context,
	action string,
	raw json.RawMessage,
) (interface{}, error) {

	// MCP 标准结构
	var req struct {
		Protocol string          `json:"protocol"`
		Params   json.RawMessage `json:"params"`
	}

	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if req.Protocol == "" {
		return nil, fmt.Errorf("missing protocol")
	}

	if len(req.Params) == 0 {
		req.Params = []byte("{}")
	}

	// 调用 route engine
	err := s.engine.Dispatch(
		ctx,
		req.Protocol,
		action,
		req.Params,
	)

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

func (s *Service) ReadResource(
	ctx context.Context,
	protocol string,
) (interface{}, error) {

	return s.engine.List(ctx, protocol)
}
func (s *Service) Schema() map[string]any {
	return map[string]any{
		"type": "object",
	}
}

func (s *Service) Dispatch(
	ctx context.Context,
	action string,
	args json.RawMessage,
) (any, error) {

	err := s.engine.Dispatch(ctx, action, args)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"status": "ok",
	}, nil
}

func (s *Service) DebugDump() {
	for name := range s.engine.modules {
		fmt.Printf("[route] loaded module: %s\n", name)
	}
}
