// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package live

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestServerStreamsEventsOverWebSocket(t *testing.T) {
	hub := NewHub()
	s := &Server{hub: hub}

	httpSrv := httptest.NewServer(http.HandlerFunc(s.handleEvents))
	defer httpSrv.Close()
	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL+"/ws/events", nil)
	if err != nil {
		t.Fatalf("websocket Dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// The handler subscribes after the upgrade; wait for it to register.
	waitFor(t, func() bool { return hub.Subscribers() == 1 })

	hub.Publish(Event{NodeID: "nod_1", Type: "PEER_CONNECTED"})

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("websocket Read: %v", err)
	}
	var got Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if got.NodeID != "nod_1" || got.Type != "PEER_CONNECTED" {
		t.Errorf("event: got %+v", got)
	}
}

func TestServerSubscriberReleasedOnDisconnect(t *testing.T) {
	hub := NewHub()
	s := &Server{hub: hub}
	httpSrv := httptest.NewServer(http.HandlerFunc(s.handleEvents))
	defer httpSrv.Close()
	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL+"/ws/events", nil)
	if err != nil {
		t.Fatalf("websocket Dial: %v", err)
	}
	waitFor(t, func() bool { return hub.Subscribers() == 1 })

	conn.Close(websocket.StatusNormalClosure, "")
	waitFor(t, func() bool { return hub.Subscribers() == 0 })
}

// waitFor polls cond for up to two seconds.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}
