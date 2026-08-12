package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

type Engine struct {
	reg   *Registry
	debug bool
}

func NewEngine(reg *Registry, debug bool) *Engine {
	engine := &Engine{
		reg:   reg,
		debug: debug,
	}

	if engine.debug {
		engine.reg.Dump()
	}
	return engine
}

func (e *Engine) ProcessMessage(ctx context.Context, payload []byte) ([]byte, error) {
	var req JSONRPCRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		resp := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error:   &RPCError{Code: ErrCodeParseError, Message: fmt.Sprintf("invalid JSON payload: %v", err)},
		}
		return json.Marshal(resp)
	}

	// 处理 JSON-RPC 通知（ID 为 nil 且定义了 Method）
	if req.ID == nil && req.Method != "" {
		e.handleNotification(ctx, &req)
		return nil, nil // 通知类消息无需发回包
	}

	var (
		result any
		rpcErr *RPCError
	)

	switch req.Method {

	// ======================
	// MCP 握手协议
	// ======================
	case "initialize":
		result = map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "mcp-go-server",
				"version": "1.0.0",
			},
		}

	case "ping":
		result = map[string]any{}

	// ======================
	// tools/list
	// ======================
	case "tools/list":
		result = map[string]any{
			"tools": e.reg.ToolList(),
		}

	// ======================
	// tools/call
	// ======================
	case "tools/call":
		var call struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}

		if err := json.Unmarshal(req.Params, &call); err != nil {
			rpcErr = &RPCError{
				Code:    ErrCodeInvalidParams,
				Message: fmt.Sprintf("invalid params: %v", err),
			}
			break
		}

		tc := NewToolContext(ctx, req.ID)

		res, err := e.reg.ExecuteTool(tc, call.Name, call.Arguments)
		if err != nil {
			rpcErr = &RPCError{
				Code:    ErrCodeInternalError,
				Message: err.Error(),
			}
			break
		}

		result = res

	default:
		rpcErr = &RPCError{
			Code:    ErrCodeMethodNotFound,
			Message: fmt.Sprintf("method not found: %s", req.Method),
		}
	}

	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
		Error:   rpcErr,
	}

	return json.Marshal(resp)
}

func (e *Engine) handleNotification(ctx context.Context, req *JSONRPCRequest) {
	switch req.Method {
	case "notifications/initialized":
		// 客户端已完成握手
	default:
		// 忽略未知通知
	}
}
