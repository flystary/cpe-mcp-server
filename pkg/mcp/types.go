package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// JSONRPCRequest 标准请求
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// JSONRPCResponse 标准响应
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

// RPCError 标准错误结构
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Resource
// ResourceDefinition
type ResourceDefinition struct {
	URI      string `json:"uri"`
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
}

// // Tool
// // ToolContext：运行上下文（可扩展）
type ToolContext struct {
	Context   context.Context
	RequestID interface{}
	User      string
	Meta      map[string]interface{}
}

// ToolHandler：统一执行函数
type ToolHandler func(ctx ToolContext, params json.RawMessage) (interface{}, error)

// ToolDefinition：MCP Tool 描述（静态 schema）
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// Tool：完整 Tool 对象（schema + handler）
type Tool struct {
	Def     ToolDefinition
	Handler ToolHandler
}

// ToolResponse：统一返回（兼容 MCP）
type ToolResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  *RPCError   `json:"error,omitempty"`
}

// Service
type Service interface {
	Name() string
	Actions() []string

	Schema() map[string]interface{}

	Execute(
		ctx context.Context,
		action string,
		args json.RawMessage,
	) (interface{}, error)
}

type ResourceProvider interface {
	ReadResource(ctx context.Context, path string) (interface{}, error)
}

// NewToolContext 创建标准上下文（推荐统一入口）
func NewToolContext(ctx context.Context, requestID any) ToolContext {
	return ToolContext{
		Context:   ctx,
		RequestID: normalizeRequestID(requestID),
		Meta:      make(map[string]any),
	}
}

// normalizeRequestID 统一 JSON-RPC ID 类型
func normalizeRequestID(id any) string {
	if id == nil {
		return ""
	}

	switch v := id.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// RPCHeader 用于提前抽离 JSON-RPC 请求的关键头信息
type RPCHeader struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
}

// IsNotification 判断是否为无需响应的 Notification 请求
func (h *RPCHeader) IsNotification() bool {
	return len(h.ID) == 0 || string(h.ID) == "null"
}

// ParseID 解析出原生 id 结构 (string, number 或 nil)
func (h *RPCHeader) ParseID() interface{} {
	if h.IsNotification() {
		return nil
	}
	var id interface{}
	_ = json.Unmarshal(h.ID, &id)
	return id
}
