package mcp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type SseServer struct {
	addr        string
	engine      *MCPEngine
	messageChan chan []byte
	sessions    map[string]chan []byte
	mu          sync.RWMutex
}

func NewSseServer(addr string, engine *MCPEngine) *SseServer {
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
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, _ := w.(http.Flusher)
	ch := make(chan []byte, 100)

	s.mu.Lock()
	s.sessions["default"] = ch
	s.mu.Unlock()

	_, _ = fmt.Fprintf(w, "event: endpoint\ndata: /message\n\n")
	flusher.Flush()

	for {
		select {
		case msg := <-ch:
			_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(msg))
			flusher.Flush()
		case <-r.Context().Done():
			s.mu.Lock()
			delete(s.sessions, "default")
			s.mu.Unlock()
			return
		}
	}
}

func (s *SseServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	payload, _ := io.ReadAll(r.Body)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	responseBytes, err := s.engine.ProcessMessage(ctx, payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.mu.RLock()
	ch, exists := s.sessions["default"]
	s.mu.RUnlock()

	if exists {
		ch <- responseBytes
		w.WriteHeader(http.StatusAccepted)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
