package mcp

import (
	"bufio"
	"context"
	"io"
	"os"
)

type CliServer struct {
	engine *MCPEngine
}

func NewCliServer(engine *MCPEngine) *CliServer {
	return &CliServer{engine: engine}
}

func (s *CliServer) Start() error {
	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		responseBytes, err := s.engine.ProcessMessage(ctx, line)
		if err != nil {
			continue
		}

		_, _ = os.Stdout.Write(responseBytes)
		_, _ = os.Stdout.Write([]byte("\n"))
	}
}
