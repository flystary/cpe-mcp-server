package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"cpe-mcp-server/internal/route"
	_ "cpe-mcp-server/internal/route/static" // auto register static
	"cpe-mcp-server/pkg/mcp"
	"cpe-mcp-server/pkg/mcp/tools"
	_ "cpe-mcp-server/pkg/mcp/tools"
)

var debug = true

func main() {
	// 定义 -m 参数，默认值为 "sse"
	modePtr := flag.String("m", "sse", "运行模式: sse (Web长连接) 或 cli (Stdio管道)")
	flag.Parse()
	mode := strings.ToLower(strings.TrimSpace(*modePtr))
	// 初始化路由引擎
	routeEngine := route.NewEngine()
	// 动态注册进引擎实例
	for name, factory := range route.Factories() {
		log.Printf("[INIT] 动态装载核心网络模块驱动: %s", name)
		routeEngine.RegisterModule(name, factory())
	}
	// 将打通的动态引擎包装进 MCP 顶层 Service 容器
	service := route.NewService(routeEngine)

	// MCP 网关注册
	reg := mcp.NewRegistry()
	reg.RegisterService("route", true, service.New)

	// 注入原子工具
	tools.Setup(reg)

	// 实例化总线消息引擎
	engine := mcp.NewEngine(reg, debug)

	switch mode {
	case "cli":

		log.SetOutput(os.Stderr)
		log.Println("[INFO] CLI mode started")

		server := mcp.NewCliServer(engine)
		if err := server.Start(); err != nil {
			log.Fatalf("CLI error: %v", err)
		}

	case "sse":
		log.Println("[INFO] SSE mode started :8080")

		server := mcp.NewSseServer(":8080", engine)
		if err := server.Start(); err != nil {
			log.Fatalf("SSE error: %v", err)
		}

	default:
		log.Printf("unknown mode: %s", mode)
		flag.Usage()
		os.Exit(1)
	}
}
