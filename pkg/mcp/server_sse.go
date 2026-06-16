package mcp

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type SseServer struct {
	addr   string
	engine *Engine

	mu sync.RWMutex
	// messageChan chan []byte
	sessions map[string]chan []byte
}

func NewSseServer(addr string, engine *Engine) *SseServer {
	return &SseServer{
		addr:     addr,
		engine:   engine,
		sessions: make(map[string]chan []byte),
	}
}

func (s *SseServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/sse", s.handleSSE)
	mux.HandleFunc("/message", s.handleMessage)
	return http.ListenAndServe(s.addr, mux)
}
func (s *SseServer) handleSSE(w http.ResponseWriter, r *http.Request) {

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream not supported", http.StatusInternalServerError)
		return
	}

	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		sessionID = "default"
	}

	ch := make(chan []byte, 64)

	s.mu.Lock()
	s.sessions[sessionID] = ch
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.sessions, sessionID)
		// close(ch)
		s.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// endpoint hint
	fmt.Fprintf(w, "event: endpoint\ndata: /message\n\n")
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {

		case msg := <-ch:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()

		case <-ticker.C:
			// heartbeat
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

func (s *SseServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	payload, _ := io.ReadAll(r.Body)

	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		sessionID = "default"
	}

	taskCtx, taskCancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer taskCancel()

	// 提前读取 session，判断会话是否仍存活
	s.mu.RLock()
	ch, active := s.sessions[sessionID]
	s.mu.RUnlock()

	if !active {
		http.Error(w, "session expired or inactive", http.StatusBadRequest)
		return
	}

	// async execute
	go func(targetChan chan []byte, ctx context.Context, cancel context.CancelFunc) {
		defer func() {
			cancel()
			if rec := recover(); rec != nil {
				log.Printf("捕获异常: %v", rec)
			}
		}()

		resp, err := s.engine.ProcessMessage(ctx, payload)
		if err != nil {
			resp = []byte(fmt.Sprintf(`{"jsonrpc":"2.0","error":{"code":500,"message":"%s"}}`, err.Error()))
		}

		s.mu.RLock()
		_, stillExist := s.sessions[sessionID]
		s.mu.RUnlock()

		if !stillExist {
			// 此时长连接已断开，会话已销毁，直接安全退出，避免向可能关闭的通道写数据
			return
		}

		select {
		case targetChan <- resp:
		case <-ctx.Done():
			// 处理超时
		default:
			// drop if slow client
		}
	}(ch, taskCtx, taskCancel)

	w.WriteHeader(http.StatusAccepted)
}
