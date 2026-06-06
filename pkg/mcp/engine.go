package mcp

import (
	"context"
	"encoding/json"
)

type MCPEngine struct{}

func NewEngine() *MCPEngine { return &MCPEngine{} }

func (e *MCPEngine) ProcessMessage(ctx context.Context, payload []byte) ([]byte, error) {
	var req JSONRPCREquest
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, err
	}

	var res interface{}
	var rpcErr *RPCError

	switch req.Method {
	case "tools/list":
		res = map[string]interface{}{"tools": GetToolList()}
	case "tools/call":
		var callArgs struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		_ = json.Unmarshal(req.Params, &callArgs)
		result, err := ExecuteTool(ctx, callArgs.Name, callArgs.Arguments)
		if err != nil {
			rpcErr = &RPCError{Code: 500, Messgae: err.Error()}
		} else {
			res = result
		}
	default:
		rpcErr = &RPCError{Code: -32601, Messgae: "Method not found"}
	}
	response := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID}
	if rpcErr != nil {
		response.Error = rpcErr
	} else {
		response.Result = res
	}

	return json.Marshal(response)

}
