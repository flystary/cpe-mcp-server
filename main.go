package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"cpe-mcp-server/internal/route"
	_ "cpe-mcp-server/internal/route/static" // auto register static
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
	routeEngine := route.NewEngine()
	reg := mcp.NewRegistry()
	reg.RegisterService(
		"route",
		true,
		func() mcp.Service {
			return route.NewService(routeEngine)
		},
	)
	engine := mcp.NewEngine(reg, true)

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
