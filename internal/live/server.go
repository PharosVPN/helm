// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package live

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

const (
	wsWriteTimeout  = 10 * time.Second
	shutdownTimeout = 5 * time.Second
)

// Server is helm's localhost admin HTTP server. For M4 it exposes the live
// event WebSocket and a health check; the SvelteKit admin UI mounts here in M5.
type Server struct {
	hub  *Hub
	http *http.Server
}

// NewServer builds a Server bound to addr (a localhost address — helm opens no
// public ports).
func NewServer(addr string, hub *Hub) *Server {
	s := &Server{hub: hub}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/events", s.handleEvents)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s.http = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s
}

// Addr returns the server's configured listen address.
func (s *Server) Addr() string { return s.http.Addr }

// Run serves until ctx is cancelled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	errc := make(chan error, 1)
	go func() {
		err := s.http.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errc <- err
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return s.http.Shutdown(shutCtx)
	}
}

// handleEvents upgrades to a WebSocket and streams live events to the browser.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	// The server binds to localhost only; M5 puts session auth in front of
	// this endpoint. Until then any local origin may connect.
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer c.CloseNow()

	// CloseRead drains and discards client frames, cancelling ctx on close.
	ctx := c.CloseRead(r.Context())

	id, events := s.hub.Subscribe()
	defer s.hub.Unsubscribe(id)

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			writeCtx, cancel := context.WithTimeout(ctx, wsWriteTimeout)
			err = c.Write(writeCtx, websocket.MessageText, data)
			cancel()
			if err != nil {
				return
			}
		}
	}
}
