package tools

import (
	"cpe-mcp-server/pkg/mcp"
	"encoding/json"
)

func init() {
	Register(
		"get_system_stats",
		"直接调取网关物理设备当前的 CPU、内存与系统运行时间快照",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"detailed": map[string]any{
					"type":        "boolean",
					"description": "是否返回每颗 CPU 核心的详细监控指标",
				},
			},
		},
		func(ctx mcp.ToolContext, params json.RawMessage) (interface{}, error) {
			// 实际业务逻辑可在此接入系统底层命令或 eBPF 监控快照
			return map[string]any{
				"cpu_usage": "14.2%",
				"mem_free":  "1240MB",
				"uptime":    "48h12m",
				"status":    "healthy",
			}, nil
		},
	)
}
