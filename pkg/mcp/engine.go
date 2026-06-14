package mcp

import (
	"context"
	"encoding/json"
)

type Engine struct {
	reg   *Registry
	debug bool
}

func NewEngine(reg *Registry, debug bool) *Engine {
	return &Engine{
		reg:   reg,
		debug: debug,
	}
}

func (e *Engine) ProcessMessage(
	ctx context.Context,
	payload []byte,
) ([]byte, error) {

	var req JSONRPCRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, err
	}

	var (
		result any
		rpcErr *RPCError
	)

	switch req.Method {

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
			rpcErr = &RPCError{-32602, err.Error(), nil}
			break
		}

		tc := NewToolContext(ctx, req.ID)

		res, err := e.reg.ExecuteTool(tc, call.Name, call.Arguments)
		if err != nil {
			rpcErr = &RPCError{500, err.Error(), nil}
			break
		}

		result = res

	default:
		rpcErr = &RPCError{-32601, "method not found", nil}
	}

	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
		Error:   rpcErr,
	}

	return json.Marshal(resp)
}
