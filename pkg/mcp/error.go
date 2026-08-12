package mcp

import "encoding/json"

// 标准 JSON-RPC 2.0 错误码
const (
	ErrCodeParseError     = -32700 // 无效的 JSON
	ErrCodeInvalidRequest = -32600 // 无效的 JSON-RPC 请求
	ErrCodeMethodNotFound = -32601 // 方法不存在
	ErrCodeInvalidParams  = -32602 // 无效的方法参数
	ErrCodeInternalError  = -32603 // 内部错误 / Panic

	// MCP / 应用自定义错误码区间 (-32000 ~ -32099)
	ErrCodeUnauthorized = -32001 // 未授权/Token无效
	ErrCodeRateLimited  = -32002 // 请求过于频繁
)

// BuildRPCError 构造标准 JSON-RPC 2.0 错误响应
func BuildRPCError(id interface{}, code int, message string) []byte {
	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	b, _ := json.Marshal(resp)
	return b
}
