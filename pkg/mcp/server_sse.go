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
		close(ch)
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

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// async execute
	go func() {

		resp, err := s.engine.ProcessMessage(ctx, payload)

		if err != nil {
			resp = []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		}

		s.mu.RLock()
		ch, ok := s.sessions[sessionID]
		s.mu.RUnlock()

		if !ok {
			return
		}

		select {
		case ch <- resp:
		default:
			// drop if slow client
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}
