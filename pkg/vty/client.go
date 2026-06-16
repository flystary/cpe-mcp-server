package vty

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

var (
	ErrTimeout        = errors.New("vty socket read timeout")
	ErrConfigRejected  = errors.New("vty engine rejected configuration commands")
)

// vtyClient
type vtyClient struct {
	socketPath string
}

// executeInternal
func (c *vtyClient) execute(ctx context.Context, commands []string) error {
	if len(commands) == 0 {
		return nil
	}

	rawOutput, err := c.roundTrip(ctx, commands, 1500*time.Millisecond)
	if err != nil {
		return err
	}

	if strings.Contains(rawOutput, "% ") || strings.Contains(rawOutput, "Ambiguous command") {
		return fmt.Errorf("%w: \n%s", ErrConfigRejected, strings.TrimSpace(rawOutput))
	}
	return nil
}

// query
func (c *vtyClient) query(ctx context.Context, command string) (string, error) {
	cmds := []string{command, "quit"}
	rawOutput, err := c.roundTrip(ctx, cmds, 2000*time.Millisecond)
	if err != nil {
		return "", err
	}
	return c.cleanNoise(rawOutput, command), nil
}

// roundTrip
func (c *vtyClient) roundTrip(ctx context.Context, commands []string, readTimeout time.Duration) (string, error) {
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return "", fmt.Errorf("vty connection failed [%s]: %w", c.socketPath, err)
	}
	defer conn.Close()

	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return "", fmt.Errorf("invalid unix domain socket assertion")
	}

	var payload bytes.Buffer
	for _, cmd := range commands {
		trimmed := strings.TrimSpace(cmd)
		if len(trimmed) > 0 {
			payload.WriteString(trimmed + "\n")
		}
	}

	if _, err = unixConn.Write(payload.Bytes()); err != nil {
		return "", fmt.Errorf("vty write failed: %w", err)
	}

	// 🚀 半关闭（CloseWrite）：发出 FIN 催促 FRR 立刻交出回显
	_ = unixConn.CloseWrite()

	var responseBuf bytes.Buffer
	replyBuffer := make([]byte, 4096)
	_ = unixConn.SetReadDeadline(time.Now().Add(readTimeout))

	for {
		n, err := unixConn.Read(replyBuffer)
		if n > 0 {
			responseBuf.Write(replyBuffer[:n])
		}
		if err != nil {
			if strings.Contains(err.Error(), "EOF") {
				break
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return "", ErrTimeout
			}
			return "", fmt.Errorf("vty read broken: %w", err)
		}
	}

	return responseBuf.String(), nil
}

func (c *vtyClient) cleanNoise(raw, cmd string) string {
	lines := strings.Split(raw, "\n")
	var cleanLines []string
	startParsing := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, cmd) {
			startParsing = true
			continue
		}
		if trimmed == "quit" || strings.HasSuffix(trimmed, "#") || strings.HasSuffix(trimmed, ">") {
			continue
		}
		if startParsing && len(trimmed) > 0 {
			cleanLines = append(cleanLines, line)
		}
	}
	if len(cleanLines) == 0 {
		return raw
	}
	return strings.Join(cleanLines, "\n")
}
