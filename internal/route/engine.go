package route

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

type Engine struct {
	mu      sync.RWMutex
	modules map[string]RouteHandler
}

// NewEngine 构造一个完全干净、无全局状态绑定的空引擎
func NewEngine() *Engine {
	return &Engine{
		modules: map[string]RouteHandler{},
	}
}

// RegisterModule 允许在初始化或程序运行时，动态挂载一个协议驱动
func (e *Engine) RegisterModule(name string, handler RouteHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.modules[name] = handler
}

// UnregisterModule 允许运行时热卸载某个协议模块
func (e *Engine) UnregisterModule(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.modules, name)
}

func (e *Engine) Dispatch(ctx context.Context, protocol string, action string, args []byte) error {

	// 找 module
	e.mu.RLock()
	mod, ok := e.modules[protocol]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("module not found: %s", protocol)
	}

	// schema check（动作是否存在）
	schema := mod.Schema()
	_, ok = schema.Actions[action]
	if !ok {
		return fmt.Errorf("unknown action: %s.%s", protocol, action)
	}

	// _ = act // 这里保留用于未来扩展（字段级校验/策略）

	// module validate（真正校验逻辑）
	validated, err := mod.Validate(action, args)
	if err != nil {
		return fmt.Errorf("validate failed: %w", err)
	}

	// 标准化 payload（避免 handler 再解析乱结构）
	payload, err := json.Marshal(validated)
	if err != nil {
		return fmt.Errorf("marshal validated args failed: %w", err)
	}

	// 执行
	return mod.Dispatch(ctx, action, payload)
}

func (e *Engine) List(ctx context.Context, protocol string) (interface{}, error) {

	e.mu.RLock()
	mod, ok := e.modules[protocol]
	e.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("module not enabled: %s", protocol)
	}

	return mod.List(ctx)
}

func (e *Engine) Schema() map[string]any {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := map[string]any{}

	for name, mod := range e.modules {
		out[name] = mod.Schema()
	}

	return out
}

// GetModulesSnapshot 获取当前已加载模块的快照（供 Service 汇总 Actions 使用）
func (e *Engine) GetModulesSnapshot() map[string]RouteHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()

	cp := make(map[string]RouteHandler)
	for k, v := range e.modules {
		cp[k] = v
	}
	return cp
}
