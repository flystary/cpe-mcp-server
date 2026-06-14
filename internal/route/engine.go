package route

import (
	"context"
	"encoding/json"
	"fmt"
)

type Engine struct {
	modules map[string]RouteHandler
}

func NewEngine() *Engine {
	return &Engine{
		modules: map[string]RouteHandler{},
	}
}
func (e *Engine) Dispatch(
	ctx context.Context,
	protocol string,
	action string,
	args []byte,
) error {

	// 1. 找 module
	mod, ok := e.modules[protocol]
	if !ok {
		return fmt.Errorf("module not found: %s", protocol)
	}

	// 2. schema check（动作是否存在）
	schema := mod.Schema()

	act, ok := schema.Actions[action]
	if !ok {
		return fmt.Errorf("unknown action: %s.%s", protocol, action)
	}

	_ = act // 这里保留用于未来扩展（字段级校验/策略）

	// 3. module validate（真正校验逻辑）
	validated, err := mod.Validate(action, args)
	if err != nil {
		return fmt.Errorf("validate failed: %w", err)
	}

	// 4. 标准化 payload（避免 handler 再解析乱结构）
	payload, err := json.Marshal(validated)
	if err != nil {
		return fmt.Errorf("marshal validated args failed: %w", err)
	}

	// 5. 执行
	return mod.Dispatch(ctx, action, payload)
}

func (e *Engine) List(
	ctx context.Context,
	protocol string,
) (interface{}, error) {

	mod, ok := e.modules[protocol]
	if !ok {
		return nil, fmt.Errorf("module not enabled: %s", protocol)
	}

	return mod.List(ctx)
}

func (e *Engine) Schema() map[string]any {
	out := map[string]any{}

	for name, mod := range e.modules {
		out[name] = mod.Schema()
	}

	return out
}
