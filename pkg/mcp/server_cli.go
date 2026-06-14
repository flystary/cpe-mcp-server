package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type CliServer struct {
	engine *Engine
}

func NewCliServer(engine *Engine) *CliServer {
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

		// trim
		raw := strings.TrimSpace(string(line))
		if raw == "" {
			continue
		}

		// recover panic
		func() {

			defer func() {
				if r := recover(); r != nil {

					errResp := JSONRPCResponse{
						JSONRPC: "2.0",
						Error: &RPCError{
							Code:    500,
							Message: fmt.Sprintf("panic: %v", r),
						},
					}

					b, _ := json.Marshal(errResp)
					_, _ = os.Stdout.Write(b)
					_, _ = os.Stdout.Write([]byte("\n"))
				}
			}()

			resp, err := s.engine.ProcessMessage(ctx, []byte(raw))
			if err != nil {

				errResp := JSONRPCResponse{
					JSONRPC: "2.0",
					Error: &RPCError{
						Code:    500,
						Message: err.Error(),
					},
				}

				b, _ := json.Marshal(errResp)
				_, _ = os.Stdout.Write(b)
				_, _ = os.Stdout.Write([]byte("\n"))

				return
			}

			_, _ = os.Stdout.Write(resp)
			_, _ = os.Stdout.Write([]byte("\n"))

		}()
	}
}
