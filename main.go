package main

import (
	"flag"
	"log"
	"os"
	"strings"

	_ "cpe-mcp-server/internal/route/static" // 触发自发现注册
	"cpe-mcp-server/pkg/mcp"
)

var (
	debug = true
)

func main() {
	// 定义 -m 参数，默认值为 "sse"
	modePtr := flag.String("m", "sse", "运行模式: sse (Web长连接) 或 cli (Stdio管道)")
	flag.Parse()
	mode := strings.ToLower(strings.TrimSpace(*modePtr))
	engine := mcp.NewEngine(debug)

	switch mode {
	case "cli":
		// 原生 Stdio 管道模式
		log.SetOutput(os.Stderr)
		log.Println("[INFO] CPE-MCP 正在以底层管道(CLI)模式拉起...")

		server := mcp.NewCliServer(engine)
		if err := server.Start(); err != nil {
			log.Fatalf("[FATAL] CLI 管道断裂: %v", err)
		}

	case "sse":
		// Web SSE 长连接模式
		log.Println("[INFO] CPE-MCP 正在以云端长连接(SSE)模式拉起...")

		server := mcp.NewSseServer(":8080", engine)
		if err := server.Start(); err != nil {
			log.Fatalf("[FATAL] SSE 端口被抢占: %v", err)
		}

	default:
		log.Printf("[ERROR] 未知的运行模式: '%s'，仅支持 'sse' 或 'cli'\n", *modePtr)
		flag.Usage()
		os.Exit(1)
	}
}
