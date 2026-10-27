package sse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestHeartbeatKeepsIdleStreamAlive(t *testing.T) {
	cfg := testConfig()
	cfg.HeartbeatInterval = 40 * time.Millisecond
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	resp, reader := openStream(t, server, url.Values{"user": {"idle"}}, nil)
	defer resp.Body.Close()
	waitUntil(t, time.Second, func() bool { return m.Online("idle") })

	found := make(chan struct{})
	go func() {
		for {
			line, err := reader.ReadString(byte(10))
			if err != nil {
				return
			}
			if strings.HasPrefix(strings.TrimSpace(line), ":") {
				close(found)
				return
			}
		}
	}()
	select {
	case <-found:
	case <-time.After(time.Second):
		t.Fatal("no heartbeat comment received")
	}
	if !m.Online("idle") {
		t.Fatal("idle stream was closed after a heartbeat")
	}
}

func TestSlowClientDropReconnectsWithReplay(t *testing.T) {
	cfg := testConfig()
	cfg.SendQueue = 1
	cfg.CloseDrainTimeout = 50 * time.Millisecond
	cfg.Replay = replayFunc(func(_ context.Context, userID, lastID string) ([]Event, error) {
		if userID != "recover" {
			return nil, fmt.Errorf("unexpected user %q", userID)
		}
		if lastID == "" {
			return nil, nil
		}
		if lastID != "e1" {
			return nil, fmt.Errorf("unexpected last id %q", lastID)
		}
		return []Event{{ID: "e2", Name: "replayed", Data: json.RawMessage(`{"n":2}`)}}, nil
	})
	m := startTestManager(t, cfg)

	writer := newBlockingWriter()
	req := httptest.NewRequest(http.MethodGet, "/events?user=recover", nil)
	done := make(chan struct{})
	go func() {
		m.ServeHTTP(writer, req)
		close(done)
	}()
	waitUntil(t, time.Second, func() bool { return m.Online("recover") })

	if sent, err := m.SendEventToUser("recover", Event{ID: "e1", Name: "one"}); err != nil || sent != 1 {
		t.Fatalf("first send = %d, %v", sent, err)
	}
	if _, err := m.SendEventToUser("recover", Event{ID: "e2", Name: "two"}); !errors.Is(err, ErrSlowClient) {
		t.Fatalf("second send = %v, want ErrSlowClient", err)
	}
	writer.release()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("first stream did not close")
	}
	data := string(writer.data())
	if !strings.Contains(data, "id: e1") {
		t.Fatalf("first stream did not deliver e1: %q", data)
	}
	if !strings.Contains(data, "1013") {
		t.Fatalf("first stream did not close with 1013: %q", data)
	}

	server := httptest.NewServer(m)
	defer server.Close()
	resp, reader := openStream(t, server, url.Values{"user": {"recover"}}, http.Header{"Last-Event-ID": {"e1"}})
	defer resp.Body.Close()
	event := readSSEEvent(t, reader)
	if event["event"] != "replayed" || event["id"] != "e2" {
		t.Fatalf("replay event = %+v", event)
	}
}
