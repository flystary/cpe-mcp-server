package main

/* 正式流转打通后的控制面调用链：
MCP Engine (ProcessMessage)
    ↓
Registry.ExecuteTool() (触发绑定的闭包 Handler)
    ↓
RouteService.Execute() (拆解路由协议 Protocol 类型，向内透传数据)
    ↓
RouteEngine.Dispatch() (通过读写锁安全调度指定协议驱动)
    ↓
Static.Validate() (驱动层对扁平参数执行强类型反序列化与安全清洗)
    ↓
Static.Dispatch() (承接校验后的 Payload 并按 Action 进行路由分流)
    ↓
Static.apply() (执行物理存储持久化，并通过 vty.Zebra 下刷内核路由表)
*/

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
