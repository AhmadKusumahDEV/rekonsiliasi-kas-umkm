package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// BenchmarkHubBroadcast1000 measures hub fanout, not network capacity. Each
// client has a drain goroutine so the benchmark is not limited by a full queue.
func BenchmarkHubBroadcast1000(b *testing.B) {
	hub := newHub()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	clients := make([]*Client, 0, 1000)
	for i := 0; i < 1000; i++ {
		recorder := httptest.NewRecorder()
		client := newClient(context.Background(), recorder, http.NewResponseController(recorder), req, hub, fmt.Sprintf("user-%d", i), clientOptions{
			sendQueue:     1024,
			maxRooms:      -1,
			drainTimeout:  time.Millisecond,
			maxEventBytes: defaultMaxEventBytes,
			writeTimeout:  time.Second,
		})
		if !hub.register(client) {
			b.Fatal("register failed")
		}
		clients = append(clients, client)
		go drainBenchmarkClient(client)
	}
	payload := json.RawMessage(`{"id":"message-1","content":"benchmark"}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := hub.Broadcast(Event{Name: "message.new", Data: payload}); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	for _, client := range clients {
		client.closeNow(1000, "benchmark complete")
	}
}

func drainBenchmarkClient(client *Client) {
	for {
		select {
		case <-client.send:
			client.pending.Add(-1)
		case <-client.done:
			return
		}
	}
}

func BenchmarkEncodeEvent(b *testing.B) {
	event := Event{
		ID:   "1",
		Name: "message.new",
		Data: json.RawMessage(`{"id":"message-1","content":"benchmark"}`),
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := EncodeEvent(event); err != nil {
			b.Fatal(err)
		}
	}
}
