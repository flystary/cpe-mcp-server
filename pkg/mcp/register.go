package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

type Service interface {
	Name() string
	ReadResource(ctx context.Context, subPath string) (interface{}, error)
	ExecuteTool(ctx context.Context, action string, args json.RawMessage) (interface{}, error)
	GetSchema() interface{}
}

type ServiceFactory func() Service

type Registry struct {
	mu           sync.RWMutex
	services     map[string]Service
	tools        map[string]ToolDefinition
	toolHandlers map[string]func(ctx context.Context, args json.RawMessage) (interface{}, error)
}

var globalRegistry = &Registry{
	services:     make(map[string]Service),
	tools:        make(map[string]ToolDefinition),
	toolHandlers: make(map[string]func(ctx context.Context, args json.RawMessage) (interface{}, error)),
}

func RegisterService(name string, enable bool, factory ServiceFactory) {
	if !enable {
		return
	}

	globalRegistry.mu.Lock()
	srv := factory()
	srvName := strings.ToLower(name)
	globalRegistry.services[srvName] = srv
	globalRegistry.mu.Unlock()

	RegisterTool(ToolDefinition{
		Name:        fmt.Sprintf("configure_%s", srvName),
		Description: fmt.Sprintf("对边缘 Linux %s 矩阵下发动态控制策略", name),
		InputSchemaFunc: func() interface{} {
			return srv.GetSchema()
		},
	}, func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		return dispatchGlobalTool(ctx, srvName, args)
	})
}

func RegisterTool(def ToolDefinition, handler func(ctx context.Context, args json.RawMessage) (interface{}, error)) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.tools[def.Name] = def
	globalRegistry.toolHandlers[def.Name] = handler
}

func GetToolList() []map[string]interface{} {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	var list []map[string]interface{}
	for _, t := range globalRegistry.tools {
		list = append(list, map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchemaFunc(),
		})
	}
	return list
}

func ExecuteTool(ctx context.Context, name string, args json.RawMessage) (interface{}, error) {
	globalRegistry.mu.RLock()
	handler, exists := globalRegistry.toolHandlers[name]
	globalRegistry.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("mcp tool executor '%s' not found or inactive", name)
	}
	return handler(ctx, args)
}

func dispatchGlobalTool(ctx context.Context, srvName string, rawArgs json.RawMessage) (interface{}, error) {
	globalRegistry.mu.RLock()
	srv, exists := globalRegistry.services[srvName]
	globalRegistry.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("service '%s' offline", srvName)
	}

	var baseArgs struct {
		Action string `json:"action"`
	}

	if err := json.Unmarshal(rawArgs, &baseArgs); err != nil {
		return nil, err
	}
	return srv.ExecuteTool(ctx, baseArgs.Action, rawArgs)
}
