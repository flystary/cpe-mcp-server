package mcp

import "encoding/json"

type JSONRPCREquest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Messgae string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ResourceDefinition struct {
	URI      string `json:"uri"`
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
}

type ToolDefinition struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	InputSchemaFunc func() interface{}
}

type ToolResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
}
