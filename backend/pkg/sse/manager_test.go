package sse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSendEventToUser(t *testing.T) {
	m := startTestManager(t, testConfig())
	server := serveTestManager(t, m)
	resp, reader := openStream(t, server, url.Values{"user": {"alice"}}, nil)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	waitUntil(t, time.Second, func() bool { return m.Online("alice") })

	sent, err := m.SendEventToUser("alice", Event{Name: "hello", Data: json.RawMessage(`{"x":1}`)})
	if err != nil || sent != 1 {
		t.Fatalf("send = %d, %v", sent, err)
	}
	event := readSSEEvent(t, reader)
	if event["event"] != "hello" || event["data"] != `{"x":1}` || event["id"] == "" {
		t.Fatalf("event = %+v", event)
	}
}

func TestRoomBroadcastAndUserOperations(t *testing.T) {
	m := startTestManager(t, testConfig())
	server := serveTestManager(t, m)
	a1, r1 := openStream(t, server, url.Values{"user": {"a"}, "room": {"r"}}, nil)
	a2, r2 := openStream(t, server, url.Values{"user": {"a"}, "room": {"r"}}, nil)
	b, rb := openStream(t, server, url.Values{"user": {"b"}}, nil)
	defer func() {
		if err := a1.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	defer func() {
		if err := a2.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	defer func() {
		if err := b.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	waitUntil(t, time.Second, func() bool {
		return m.Stats().ConnectedClients == 3 && m.hub.RoomClientCount("r") == 2
	})

	if sent, err := m.BroadcastToRoom("r", Event{Name: "room", Data: json.RawMessage(`{"n":1}`)}); err != nil || sent != 2 {
		t.Fatalf("room broadcast = %d, %v", sent, err)
	}
	for name, reader := range map[string]*bufio.Reader{"a1": r1, "a2": r2} {
		if event := readSSEEvent(t, reader); event["event"] != "room" {
			t.Fatalf("%s room event = %+v", name, event)
		}
	}

	if joined := m.JoinUser("b", "r"); joined != 1 {
		t.Fatalf("JoinUser = %d, want 1", joined)
	}
	if sent, err := m.BroadcastToRoom("r", Event{Name: "after.join"}); err != nil || sent != 3 {
		t.Fatalf("broadcast after join = %d, %v", sent, err)
	}
	for name, reader := range map[string]*bufio.Reader{"a1": r1, "a2": r2, "b": rb} {
		if event := readSSEEvent(t, reader); event["event"] != "after.join" {
			t.Fatalf("%s join event = %+v", name, event)
		}
	}

	if removed := m.LeaveUser("b", "r"); removed != 1 {
		t.Fatalf("LeaveUser = %d, want 1", removed)
	}
	if sent, err := m.BroadcastToRoom("r", Event{Name: "after.leave"}); err != nil || sent != 2 {
		t.Fatalf("broadcast after leave = %d, %v", sent, err)
	}
	for _, reader := range []*bufio.Reader{r1, r2} {
		if event := readSSEEvent(t, reader); event["event"] != "after.leave" {
			t.Fatalf("leave broadcast = %+v", event)
		}
	}
}

func TestCloseRoomSendsCloseEvent(t *testing.T) {
	m := startTestManager(t, testConfig())
	server := serveTestManager(t, m)
	resp, reader := openStream(t, server, url.Values{"user": {"room-user"}, "room": {"closing"}}, nil)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	waitUntil(t, time.Second, func() bool { return m.hub.RoomClientCount("closing") == 1 })

	if closed := m.CloseRoom("closing", 4000, "room closed"); closed != 1 {
		t.Fatalf("CloseRoom = %d, want 1", closed)
	}
	event := readSSEEvent(t, reader)
	if event["event"] != "server_close" {
		t.Fatalf("close event = %+v", event)
	}
	var payload struct {
		Code   int    `json:"code"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(event["data"]), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != 4000 || payload.Reason != "room closed" {
		t.Fatalf("close payload = %+v", payload)
	}
}

func TestReplayWithLastEventID(t *testing.T) {
	cfg := testConfig()
	cfg.Replay = replayFunc(func(_ context.Context, userID, lastID string) ([]Event, error) {
		if userID != "replay-user" || lastID != "5" {
			return nil, errors.New("unexpected replay arguments")
		}
		return []Event{{ID: "6", Name: "missed", Data: json.RawMessage(`{"n":6}`)}}, nil
	})
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	resp, reader := openStream(t, server, url.Values{"user": {"replay-user"}}, http.Header{"Last-Event-ID": {"5"}})
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	event := readSSEEvent(t, reader)
	if event["id"] != "6" || event["event"] != "missed" {
		t.Fatalf("replay event = %+v", event)
	}
}

func TestReplayFailureSendsResync(t *testing.T) {
	cfg := testConfig()
	cfg.Replay = replayFunc(func(context.Context, string, string) ([]Event, error) {
		return nil, errors.New("replay unavailable")
	})
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	resp, reader := openStream(t, server, url.Values{"user": {"replay-error"}}, http.Header{"Last-Event-ID": {"1"}})
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	if event := readSSEEvent(t, reader); event["event"] != "resync" {
		t.Fatalf("replay error event = %+v", event)
	}
}

func TestReplayTruncationSendsResync(t *testing.T) {
	cfg := testConfig()
	cfg.MaxReplayEvents = 1
	cfg.Replay = replayFunc(func(context.Context, string, string) ([]Event, error) {
		return []Event{
			{ID: "1", Name: "first"},
			{ID: "2", Name: "second"},
		}, nil
	})
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	resp, reader := openStream(t, server, url.Values{"user": {"replay-limit"}}, http.Header{"Last-Event-ID": {"0"}})
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	if event := readSSEEvent(t, reader); event["event"] != "first" {
		t.Fatalf("first replay event = %+v", event)
	}
	if event := readSSEEvent(t, reader); event["event"] != "resync" {
		t.Fatalf("truncated replay event = %+v", event)
	}
}

func TestReplayOversizeSendsResync(t *testing.T) {
	cfg := testConfig()
	cfg.MaxEventBytes = 80
	cfg.Replay = replayFunc(func(context.Context, string, string) ([]Event, error) {
		return []Event{{ID: "1", Name: "large", Data: json.RawMessage(`{"blob":"` + strings.Repeat("x", 200) + `"}`)}}, nil
	})
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	resp, reader := openStream(t, server, url.Values{"user": {"replay-large"}}, http.Header{"Last-Event-ID": {"0"}})
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	if event := readSSEEvent(t, reader); event["event"] != "resync" {
		t.Fatalf("oversize replay event = %+v", event)
	}
}

func TestReplayRequiresApplicationEventID(t *testing.T) {
	cfg := testConfig()
	cfg.Replay = replayFunc(func(context.Context, string, string) ([]Event, error) {
		return nil, nil
	})
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	resp, _ := openStream(t, server, url.Values{"user": {"replay-id"}}, http.Header{"Last-Event-ID": {"1"}})
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	waitUntil(t, time.Second, func() bool { return m.Online("replay-id") })
	if _, err := m.SendEventToUser("replay-id", Event{Name: "missing.id"}); !errors.Is(err, ErrMissingEventID) {
		t.Fatalf("missing ID error = %v", err)
	}
}

func TestLastEventIDLengthLimit(t *testing.T) {
	cfg := testConfig()
	cfg.MaxLastEventIDBytes = 4
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/?user=last-id", nil)
	req.Header.Set("Last-Event-ID", "12345")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestCORSPreflight(t *testing.T) {
	cfg := testConfig()
	cfg.AllowedOrigins = []string{"https://app.example"}
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	req, _ := http.NewRequest(http.MethodOptions, server.URL, nil)
	req.Header.Set("Origin", "https://app.example")
	req.Header.Set("Access-Control-Request-Headers", "Last-Event-ID, Authorization")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("preflight status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Fatalf("allow origin = %q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Last-Event-ID") {
		t.Fatalf("allow headers = %q", got)
	}
}

func TestPresenceRefreshOnHeartbeat(t *testing.T) {
	cfg := testConfig()
	cfg.HeartbeatInterval = 20 * time.Millisecond
	presence := &recordingPresence{online: true}
	cfg.Presence = presence
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	resp, _ := openStream(t, server, url.Values{"user": {"presence-refresh"}}, nil)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	waitUntil(t, 2*time.Second, func() bool { return presence.refreshed.Load() > 0 })
}

func TestTokenExpiryAndConnectionAge(t *testing.T) {
	t.Run("token expiry", func(t *testing.T) {
		cfg := testConfig()
		cfg.Authenticate = func(*http.Request) (Identity, error) {
			return Identity{UserID: "expiring", ExpiresAt: time.Now().Add(80 * time.Millisecond)}, nil
		}
		m := startTestManager(t, cfg)
		server := serveTestManager(t, m)
		resp, reader := openStream(t, server, nil, nil)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("close response: %v", err)
			}
		}()
		assertCloseCode(t, readSSEEvent(t, reader), CloseTokenExpired)
	})

	t.Run("connection age", func(t *testing.T) {
		cfg := testConfig()
		cfg.MaxConnectionAge = 80 * time.Millisecond
		m := startTestManager(t, cfg)
		server := serveTestManager(t, m)
		resp, reader := openStream(t, server, url.Values{"user": {"age"}}, nil)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("close response: %v", err)
			}
		}()
		assertCloseCode(t, readSSEEvent(t, reader), CloseConnectionAge)
	})
}

func TestEncodeEventRejectsInjection(t *testing.T) {
	frame, err := EncodeEvent(Event{ID: "1", Name: "x", Data: json.RawMessage("{\r\n\"x\":1}")})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(frame, []byte("\r")) {
		t.Fatalf("encoded frame contains a bare CR: %q", frame)
	}
	if _, err := EncodeEvent(Event{Name: "x", Data: json.RawMessage("not json")}); err == nil {
		t.Fatal("invalid RawMessage was accepted")
	}
	comment := EncodeComment("a\rb")
	if bytes.Contains(comment, []byte("\r")) {
		t.Fatalf("comment contains a bare CR: %q", comment)
	}
}

func TestCloseRetryJitter(t *testing.T) {
	cfg := testConfig()
	cfg.Retry = 100 * time.Millisecond
	m := startTestManager(t, cfg)
	for i := 0; i < 20; i++ {
		restart := m.closeRetry(CloseServiceRestart)
		if restart < 100*time.Millisecond || restart > 200*time.Millisecond {
			t.Fatalf("restart retry = %s, want 100-200ms", restart)
		}
		age := m.closeRetry(CloseConnectionAge)
		if age < 90*time.Millisecond || age > 110*time.Millisecond {
			t.Fatalf("age retry = %s, want 90-110ms", age)
		}
	}
}

func TestShutdownUsesRestartCode(t *testing.T) {
	m := startTestManager(t, testConfig())
	server := serveTestManager(t, m)
	resp, reader := openStream(t, server, url.Values{"user": {"shutdown"}}, nil)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	waitUntil(t, time.Second, func() bool { return m.Online("shutdown") })

	errCh := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		errCh <- m.Shutdown(ctx)
	}()
	event := readSSEEvent(t, reader)
	if event["event"] != "server_close" {
		t.Fatalf("shutdown event = %+v", event)
	}
	var payload struct {
		Code    int   `json:"code"`
		RetryMS int64 `json:"retry_ms"`
	}
	if err := json.Unmarshal([]byte(event["data"]), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != CloseServiceRestart {
		t.Fatalf("shutdown code = %d, want %d", payload.Code, CloseServiceRestart)
	}
	if payload.RetryMS < 1000 || payload.RetryMS > 2000 {
		t.Fatalf("shutdown retry_ms = %d, want randomized 1000-2000", payload.RetryMS)
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestShutdownWithHTTPServerRegisterOnShutdown(t *testing.T) {
	m := startTestManager(t, testConfig())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: m}
	server.RegisterOnShutdown(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = m.Shutdown(ctx)
	})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	req, _ := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/?user=server-shutdown", nil)
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	waitUntil(t, time.Second, func() bool { return m.Online("server-shutdown") })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- server.Shutdown(ctx) }()
	assertCloseCode(t, readSSEEvent(t, bufio.NewReader(resp.Body)), CloseServiceRestart)
	if err := <-shutdownDone; err != nil {
		t.Fatal(err)
	}
}

func TestCloseDrainsQueuedEvent(t *testing.T) {
	cfg := testConfig()
	clientCh := make(chan *Client, 1)
	cfg.OnConnect = func(_ context.Context, client *Client) error {
		clientCh <- client
		return nil
	}
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	resp, reader := openStream(t, server, url.Values{"user": {"drain"}}, nil)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	waitUntil(t, time.Second, func() bool { return m.Online("drain") })
	var client *Client
	select {
	case client = <-clientCh:
	case <-time.After(time.Second):
		t.Fatal("OnConnect did not receive client")
	}
	if err := client.SendEvent("last", map[string]string{"ok": "yes"}); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if event := readSSEEvent(t, reader); event["event"] != "last" {
		t.Fatalf("drained event = %+v", event)
	}
	if event := readSSEEvent(t, reader); event["event"] != "server_close" {
		t.Fatalf("close event = %+v", event)
	}
}

func TestMiddlewareOrder(t *testing.T) {
	m := startTestManager(t, testConfig())
	var mu sync.Mutex
	var order []string
	record := func(value string) {
		mu.Lock()
		order = append(order, value)
		mu.Unlock()
	}
	wrap := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				record(name + "-in")
				next.ServeHTTP(w, r)
				record(name + "-out")
			})
		}
	}
	if err := m.Use(wrap("outer"), wrap("inner")); err != nil {
		t.Fatal(err)
	}
	if err := m.Use(nil); !errors.Is(err, ErrNilMiddleware) {
		t.Fatalf("nil middleware = %v", err)
	}

	server := serveTestManager(t, m)
	resp, _ := openStream(t, server, url.Values{"user": {"middleware"}}, nil)
	resp.Body.Close()
	waitUntil(t, time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(order) == 4
	})
	mu.Lock()
	got := strings.Join(order, ",")
	mu.Unlock()
	if got != "outer-in,inner-in,inner-out,outer-out" {
		t.Fatalf("middleware order = %q", got)
	}
}

func TestPerTypeLimitAndOnPublish(t *testing.T) {
	type observation struct {
		name     string
		duration time.Duration
		err      error
	}
	cfg := testConfig()
	cfg.EventLimits = map[string]EventLimit{
		"small": {MaxDataBytes: 5},
	}
	observations := make(chan observation, 2)
	cfg.OnPublish = func(_ context.Context, event Event, duration time.Duration, err error) {
		observations <- observation{name: event.Name, duration: duration, err: err}
	}
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	resp, reader := openStream(t, server, url.Values{"user": {"type-limit"}}, nil)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	waitUntil(t, time.Second, func() bool { return m.Online("type-limit") })

	if _, err := m.SendEventToUser("type-limit", Event{Name: "small", Data: json.RawMessage(`{"x":"123456"}`)}); !errors.Is(err, ErrEventTooLarge) {
		t.Fatalf("oversized event error = %v", err)
	}
	first := <-observations
	if first.name != "small" || first.err == nil || first.duration < 0 {
		t.Fatalf("oversized observation = %+v", first)
	}

	if _, err := m.SendEventToUser("type-limit", Event{Name: "small", Data: json.RawMessage(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if event := readSSEEvent(t, reader); event["event"] != "small" {
		t.Fatalf("small event = %+v", event)
	}
	second := <-observations
	if second.err != nil || second.duration < 0 {
		t.Fatalf("successful observation = %+v", second)
	}
}

func TestHandshakeSlotIsReleasedBeforeStreamLifetime(t *testing.T) {
	cfg := testConfig()
	cfg.MaxConcurrentHandshakes = 1
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	first, _ := openStream(t, server, url.Values{"user": {"one"}}, nil)
	defer func() {
		if err := first.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	second, _ := openStream(t, server, url.Values{"user": {"two"}}, nil)
	defer func() {
		if err := second.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	waitUntil(t, time.Second, func() bool { return m.Stats().ConnectedClients == 2 })
}

func TestBusAndPresenceHooks(t *testing.T) {
	bus := &recordingBus{}
	presence := &recordingPresence{online: true}
	cfg := testConfig()
	cfg.Bus = bus
	cfg.Presence = presence
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	resp, reader := openStream(t, server, url.Values{"user": {"bus-user"}}, nil)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	waitUntil(t, time.Second, func() bool {
		return m.Online("bus-user") && presence.added.Load() == 1
	})

	if err := m.PublishEventToUser(context.Background(), "bus-user", Event{Name: "bus.event"}); err != nil {
		t.Fatal(err)
	}
	bus.mu.Lock()
	if len(bus.published) != 1 || bus.published[0].Kind != BusUser {
		t.Fatalf("bus publish = %+v", bus.published)
	}
	handler := bus.handler
	bus.mu.Unlock()
	if handler == nil {
		t.Fatal("bus Subscribe did not register a handler")
	}
	handler(BusMessage{Kind: BusUser, UserID: "bus-user", Event: Event{Name: "bus.local"}})
	if event := readSSEEvent(t, reader); event["event"] != "bus.local" {
		t.Fatalf("bus local delivery = %+v", event)
	}
	if online, err := m.IsOnline(context.Background(), "remote-user"); err != nil || !online {
		t.Fatalf("presence online = %v, %v", online, err)
	}

	resp.Body.Close()
	waitUntil(t, time.Second, func() bool { return presence.removed.Load() == 1 })
}

func TestAuthenticationStatusAndOrigin(t *testing.T) {
	t.Run("auth error is 401", func(t *testing.T) {
		cfg := testConfig()
		cfg.Authenticate = func(*http.Request) (Identity, error) {
			return Identity{}, NewAuthError(http.StatusUnauthorized, "invalid token")
		}
		m := startTestManager(t, cfg)
		server := serveTestManager(t, m)
		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("close response: %v", err)
			}
		}()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("infrastructure error is 500", func(t *testing.T) {
		cfg := testConfig()
		cfg.Authenticate = func(*http.Request) (Identity, error) {
			return Identity{}, errors.New("database down")
		}
		m := startTestManager(t, cfg)
		server := serveTestManager(t, m)
		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("close response: %v", err)
			}
		}()
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("cross origin rejected", func(t *testing.T) {
		cfg := testConfig()
		m := startTestManager(t, cfg)
		server := serveTestManager(t, m)
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		req.Header.Set("Origin", "https://evil.example")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("close response: %v", err)
			}
		}()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("auth timeout is 503", func(t *testing.T) {
		cfg := testConfig()
		cfg.AuthTimeout = 40 * time.Millisecond
		cfg.Authenticate = func(r *http.Request) (Identity, error) {
			<-r.Context().Done()
			return Identity{}, r.Context().Err()
		}
		m := startTestManager(t, cfg)
		server := serveTestManager(t, m)
		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("close response: %v", err)
			}
		}()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("auth timeout is enforced when callback ignores context", func(t *testing.T) {
		cfg := testConfig()
		cfg.AuthTimeout = 40 * time.Millisecond
		release := make(chan struct{})
		cfg.Authenticate = func(*http.Request) (Identity, error) {
			<-release // deliberately ignores cancellation until the test releases it
			return Identity{}, nil
		}
		m := startTestManager(t, cfg)
		server := serveTestManager(t, m)
		start := time.Now()
		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("close response: %v", err)
			}
		}()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("status = %d", resp.StatusCode)
		}
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Fatalf("auth timeout took %s", elapsed)
		}
		close(release)
	})
}

func TestOnRejectReasonsAndRetryAfter(t *testing.T) {
	cfg := testConfig()
	reasons := make(chan RejectReason, 4)
	cfg.OnReject = func(_ context.Context, _ *http.Request, reason RejectReason, _ error) {
		reasons <- reason
	}
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)

	req, _ := http.NewRequest(http.MethodGet, server.URL+"?user=x", nil)
	req.Header.Set("Origin", "https://evil.example")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("origin status = %d", resp.StatusCode)
	}
	if reason := <-reasons; reason != RejectOrigin {
		t.Fatalf("origin reject reason = %q", reason)
	}

	cfg2 := testConfig()
	cfg2.MaxConnections = 1
	limitReasons := make(chan RejectReason, 1)
	cfg2.OnReject = func(_ context.Context, _ *http.Request, reason RejectReason, _ error) {
		limitReasons <- reason
	}
	m2 := startTestManager(t, cfg2)
	server2 := serveTestManager(t, m2)
	first, _ := openStream(t, server2, url.Values{"user": {"one"}}, nil)
	defer func() {
		if err := first.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	waitUntil(t, time.Second, func() bool { return m2.Stats().ConnectedClients == 1 })
	resp2, err := http.Get(server2.URL + "/?user=two")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp2.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	if resp2.StatusCode != http.StatusTooManyRequests || resp2.Header.Get("Retry-After") != "1" {
		t.Fatalf("limit status/retry = %d/%q", resp2.StatusCode, resp2.Header.Get("Retry-After"))
	}
	if reason := <-limitReasons; reason != RejectConnectionLimit {
		t.Fatalf("limit reject reason = %q", reason)
	}
}

func TestOnDisconnectGetsCloseInfoAndSkipsRejectedSetup(t *testing.T) {
	cfg := testConfig()
	type closeInfo struct {
		code   int
		reason string
	}
	disconnected := make(chan closeInfo, 1)
	cfg.OnDisconnect = func(_ context.Context, client *Client) {
		code, reason := client.CloseInfo()
		disconnected <- closeInfo{code: code, reason: reason}
	}
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	resp, _ := openStream(t, server, url.Values{"user": {"close-info"}, "room": {"closing"}}, nil)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	waitUntil(t, time.Second, func() bool { return m.hub.RoomClientCount("closing") == 1 })
	m.CloseRoom("closing", 4000, "room closed")
	select {
	case info := <-disconnected:
		if info.code != 4000 || info.reason != "room closed" {
			t.Fatalf("disconnect info = %+v", info)
		}
	case <-time.After(time.Second):
		t.Fatal("OnDisconnect was not called")
	}

	cfg2 := testConfig()
	cfg2.OnConnect = func(context.Context, *Client) error { return errors.New("nope") }
	cfg2.OnDisconnect = func(context.Context, *Client) {
		t.Error("OnDisconnect called for rejected OnConnect")
	}
	m2 := startTestManager(t, cfg2)
	server2 := serveTestManager(t, m2)
	resp2, err := http.Get(server2.URL + "/?user=rejected")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp2.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	if resp2.StatusCode != http.StatusForbidden {
		t.Fatalf("OnConnect rejection status = %d", resp2.StatusCode)
	}
	time.Sleep(30 * time.Millisecond)
}

func TestStreamingUnsupportedWriter(t *testing.T) {
	m := startTestManager(t, testConfig())
	recorder := httptest.NewRecorder()
	w := noFlushWriter{ResponseWriter: recorder}
	req := httptest.NewRequest(http.MethodGet, "/events?user=unsupported", nil)
	m.ServeHTTP(w, req)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}

func TestOnDisconnectRunsWhenHeadersFailAfterOnConnect(t *testing.T) {
	cfg := testConfig()
	connected := make(chan struct{}, 1)
	disconnected := make(chan struct{}, 1)
	cfg.OnConnect = func(context.Context, *Client) error {
		connected <- struct{}{}
		return nil
	}
	cfg.OnDisconnect = func(context.Context, *Client) {
		disconnected <- struct{}{}
	}
	m := startTestManager(t, cfg)
	recorder := httptest.NewRecorder()
	w := noFlushWriter{ResponseWriter: recorder}
	req := httptest.NewRequest(http.MethodGet, "/events?user=headers-fail", nil)
	m.ServeHTTP(w, req)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	select {
	case <-connected:
	case <-time.After(time.Second):
		t.Fatal("OnConnect was not called")
	}
	select {
	case <-disconnected:
	case <-time.After(time.Second):
		t.Fatal("OnDisconnect was not called after header failure")
	}
}

func TestConnectionLimits(t *testing.T) {
	t.Run("total", func(t *testing.T) {
		cfg := testConfig()
		cfg.MaxConnections = 1
		m := startTestManager(t, cfg)
		server := serveTestManager(t, m)
		first, _ := openStream(t, server, url.Values{"user": {"one"}}, nil)
		defer func() {
			if err := first.Body.Close(); err != nil {
				t.Errorf("close response: %v", err)
			}
		}()
		waitUntil(t, time.Second, func() bool { return m.Stats().ConnectedClients == 1 })
		resp, err := http.Get(server.URL + "/?user=two")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("close response: %v", err)
			}
		}()
		if resp.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want 429", resp.StatusCode)
		}
	})

	t.Run("per user", func(t *testing.T) {
		cfg := testConfig()
		cfg.MaxConnectionsPerUser = 1
		m := startTestManager(t, cfg)
		server := serveTestManager(t, m)
		first, _ := openStream(t, server, url.Values{"user": {"same"}}, nil)
		defer func() {
			if err := first.Body.Close(); err != nil {
				t.Errorf("close response: %v", err)
			}
		}()
		waitUntil(t, time.Second, func() bool { return m.Stats().ConnectedClients == 1 })
		resp, err := http.Get(server.URL + "/?user=same")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("close response: %v", err)
			}
		}()
		if resp.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want 429", resp.StatusCode)
		}
	})

	t.Run("rooms are trimmed", func(t *testing.T) {
		cfg := testConfig()
		cfg.MaxRoomsPerClient = 1
		m := startTestManager(t, cfg)
		server := serveTestManager(t, m)
		resp, _ := openStream(t, server, url.Values{"user": {"rooms"}, "room": {"r1", "r2"}}, nil)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("close response: %v", err)
			}
		}()
		waitUntil(t, time.Second, func() bool { return m.Stats().ConnectedClients == 1 })
		clients := m.hub.Clients()
		if len(clients) != 1 || len(clients[0].Rooms()) != 1 {
			t.Fatalf("trimmed rooms = %+v", clients)
		}
	})
}

func TestClientContextAndDisconnectHook(t *testing.T) {
	cfg := testConfig()
	clientCh := make(chan *Client, 1)
	disconnected := make(chan struct{}, 1)
	cfg.OnConnect = func(_ context.Context, client *Client) error {
		clientCh <- client
		return nil
	}
	cfg.OnDisconnect = func(_ context.Context, _ *Client) {
		disconnected <- struct{}{}
	}
	m := startTestManager(t, cfg)
	server := serveTestManager(t, m)
	resp, _ := openStream(t, server, url.Values{"user": {"context"}}, nil)
	waitUntil(t, time.Second, func() bool { return m.Online("context") })
	var client *Client
	select {
	case client = <-clientCh:
	case <-time.After(time.Second):
		t.Fatal("OnConnect did not receive client")
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-client.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("Client.Context was not canceled")
	}
	select {
	case <-disconnected:
	case <-time.After(time.Second):
		t.Fatal("OnDisconnect was not called")
	}
}

func TestSlowClientDropHook(t *testing.T) {
	hub := newHub()
	hub.slowLimit = 2
	hub.slowWindow = time.Second
	hub.queueFullPolicy = QueueFullDrop
	var drops atomic.Int64
	hub.onDrop = func(_ context.Context, _ *Client, reason DropReason) {
		if reason == DropOutboundQueueFull {
			drops.Add(1)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	recorder := httptest.NewRecorder()
	client := newClient(context.Background(), recorder, http.NewResponseController(recorder), req, hub, "slow", clientOptions{
		sendQueue: 1, maxRooms: -1, drainTimeout: 10 * time.Millisecond, maxEventBytes: defaultMaxEventBytes, writeTimeout: time.Second,
	})
	if !hub.register(client) {
		t.Fatal("register failed")
	}
	if err := client.SendEvent("queued", nil); err != nil {
		t.Fatal(err)
	}
	hub.Broadcast(Event{Name: "drop-1"})
	if client.closing.Load() {
		t.Fatal("client closed before threshold")
	}
	hub.Broadcast(Event{Name: "drop-2"})
	select {
	case <-client.Done():
	case <-time.After(time.Second):
		t.Fatal("slow client was not closed")
	}
	if drops.Load() < 2 {
		t.Fatalf("drop hook = %d, want at least 2", drops.Load())
	}
}

func TestQueueFullClosesOnFirstDrop(t *testing.T) {
	hub := newHub()
	hub.slowLimit = 3
	hub.slowWindow = time.Second
	var drops atomic.Int64
	hub.onDrop = func(_ context.Context, _ *Client, reason DropReason) {
		if reason == DropOutboundQueueFull {
			drops.Add(1)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	recorder := httptest.NewRecorder()
	client := newClient(context.Background(), recorder, http.NewResponseController(recorder), req, hub, "gap", clientOptions{
		sendQueue: 1, maxRooms: -1, drainTimeout: 10 * time.Millisecond, maxEventBytes: defaultMaxEventBytes, writeTimeout: time.Second,
	})
	if !hub.register(client) {
		t.Fatal("register failed")
	}
	if err := client.SendEvent("queued", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := hub.Broadcast(Event{ID: "e1", Name: "drop"}); !errors.Is(err, ErrSlowClient) {
		t.Fatalf("first broadcast = %v, want ErrSlowClient", err)
	}
	waitUntil(t, time.Second, func() bool { return client.gapped.Load() })
	if sent, err := hub.Broadcast(Event{ID: "e2", Name: "after"}); sent != 0 || !errors.Is(err, ErrClientClosed) {
		t.Fatalf("after-drop broadcast = %d, %v; want 0, ErrClientClosed", sent, err)
	}
	select {
	case <-client.Done():
	case <-time.After(time.Second):
		t.Fatal("gapped client was not closed")
	}
	if drops.Load() != 1 {
		t.Fatalf("drop hook = %d, want 1", drops.Load())
	}
	if !client.gapped.Load() {
		t.Fatal("client was not marked gapped")
	}
}

func TestQueueFullGapSetBeforeOnDrop(t *testing.T) {
	hub := newHub()
	hub.slowLimit = 3
	hub.slowWindow = time.Second
	started := make(chan struct{})
	release := make(chan struct{})
	hub.onDrop = func(_ context.Context, _ *Client, reason DropReason) {
		if reason != DropOutboundQueueFull {
			return
		}
		close(started)
		<-release
	}
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	recorder := httptest.NewRecorder()
	client := newClient(context.Background(), recorder, http.NewResponseController(recorder), req, hub, "blocked-drop", clientOptions{
		sendQueue: 1, maxRooms: -1, drainTimeout: 10 * time.Millisecond, maxEventBytes: defaultMaxEventBytes, writeTimeout: time.Second,
	})
	if !hub.register(client) {
		t.Fatal("register failed")
	}
	if err := client.SendEvent("queued", nil); err != nil {
		t.Fatal(err)
	}
	first := make(chan error, 1)
	go func() {
		_, err := hub.Broadcast(Event{ID: "e1", Name: "drop"})
		first <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("OnDrop did not start")
	}
	if sent, err := hub.Broadcast(Event{ID: "e2", Name: "after"}); sent != 0 || !errors.Is(err, ErrClientClosed) {
		t.Fatalf("broadcast while OnDrop is blocked = %d, %v; want 0, ErrClientClosed", sent, err)
	}
	close(release)
	if err := <-first; !errors.Is(err, ErrSlowClient) {
		t.Fatalf("first broadcast = %v, want ErrSlowClient", err)
	}
	select {
	case <-client.Done():
	case <-time.After(time.Second):
		t.Fatal("gapped client was not closed")
	}
}

func TestDirectSendClosesOnFullQueue(t *testing.T) {
	hub := newHub()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	recorder := httptest.NewRecorder()
	client := newClient(context.Background(), recorder, http.NewResponseController(recorder), req, hub, "direct", clientOptions{
		sendQueue: 1, maxRooms: -1, drainTimeout: 10 * time.Millisecond, maxEventBytes: defaultMaxEventBytes, writeTimeout: time.Second,
	})
	if !hub.register(client) {
		t.Fatal("register failed")
	}
	if err := client.SendEvent("queued", nil); err != nil {
		t.Fatal(err)
	}
	if err := client.SendEvent("drop", nil); !errors.Is(err, ErrSlowClient) {
		t.Fatalf("direct send = %v, want ErrSlowClient", err)
	}
	if !client.gapped.Load() {
		t.Fatal("direct send did not mark the client gapped")
	}
	select {
	case <-client.Done():
	case <-time.After(time.Second):
		t.Fatal("direct send did not start the close")
	}
}

func TestSlowClientStreamClosesWith1013(t *testing.T) {
	cfg := testConfig()
	cfg.SendQueue = 1
	cfg.SlowClientLimit = 1
	cfg.SlowClientWindow = time.Second
	cfg.CloseDrainTimeout = 100 * time.Millisecond
	closed := make(chan int, 1)
	cfg.OnDisconnect = func(_ context.Context, client *Client) {
		code, _ := client.CloseInfo()
		closed <- code
	}
	m := startTestManager(t, cfg)
	writer := newBlockingWriter()
	req := httptest.NewRequest(http.MethodGet, "/events?user=slow-e2e", nil)
	go m.ServeHTTP(writer, req)
	waitUntil(t, time.Second, func() bool { return m.Online("slow-e2e") })

	for i := 0; i < 3; i++ {
		_, _ = m.SendEventToUser("slow-e2e", Event{ID: fmt.Sprintf("slow-%d", i), Name: "slow"})
	}
	writer.release()
	select {
	case code := <-closed:
		if code != CloseSlowClient {
			t.Fatalf("disconnect code = %d, want %d", code, CloseSlowClient)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("slow stream did not close: stats=%+v data=%q", m.Stats(), writer.data())
	}
	data := writer.data()
	if !bytes.Contains(data, []byte("server_close")) || !bytes.Contains(data, []byte("1013")) {
		t.Fatalf("stream close frame = %q", data)
	}
}

func TestConfigValidation(t *testing.T) {
	if err := (Config{}).Validate(); err == nil {
		t.Fatal("missing Authenticate was accepted")
	}
	cfg := testConfig()
	cfg.SendQueue = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("negative SendQueue was accepted")
	}
	cfg = testConfig()
	cfg.ShutdownCloseCode = 1006
	if err := cfg.Validate(); err == nil {
		t.Fatal("invalid shutdown code was accepted")
	}
	if err := testConfig().Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	cfg = testConfig()
	cfg.AllowedOrigins = []string{"*"}
	cfg.AllowCredentials = true
	if err := cfg.Validate(); err == nil {
		t.Fatal("wildcard origin with credentials was accepted")
	}
	cfg = testConfig()
	cfg.MaxReplayEvents = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("negative MaxReplayEvents was accepted")
	}
	cfg = testConfig()
	cfg.QueueFullPolicy = "bogus"
	if err := cfg.Validate(); err == nil {
		t.Fatal("invalid QueueFullPolicy was accepted")
	}
	cfg = testConfig()
	cfg.QueueFullPolicy = QueueFullDrop
	if err := cfg.Validate(); err != nil {
		t.Fatalf("QueueFullDrop rejected: %v", err)
	}
	cfg = testConfig()
	cfg.ControlTimeout = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("negative ControlTimeout was accepted")
	}
}

func testConfig() Config {
	return Config{
		Authenticate: func(r *http.Request) (Identity, error) {
			return Identity{
				UserID: r.URL.Query().Get("user"),
				Rooms:  r.URL.Query()["room"],
			}, nil
		},
		Retry:                   time.Second,
		HeartbeatInterval:       100 * time.Millisecond,
		WriteTimeout:            time.Second,
		SendQueue:               16,
		CloseDrainTimeout:       time.Second,
		MaxEventBytes:           1 << 16,
		AuthTimeout:             time.Second,
		ConnectTimeout:          time.Second,
		CloseTimeout:            time.Second,
		ShutdownTimeout:         2 * time.Second,
		MaxConnections:          -1,
		MaxConnectionsPerUser:   -1,
		MaxRoomsPerClient:       -1,
		MaxConcurrentHandshakes: 16,
		SlowClientLimit:         3,
		SlowClientWindow:        time.Second,
	}
}

func startTestManager(t *testing.T, cfg Config) *Manager {
	t.Helper()
	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = m.Shutdown(ctx)
	})
	return m
}

func serveTestManager(t *testing.T, m *Manager) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(m)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = m.Shutdown(ctx)
		server.Close()
	})
	return server
}

func openStream(t *testing.T, server *httptest.Server, query url.Values, headers http.Header) (*http.Response, *bufio.Reader) {
	t.Helper()
	u := server.URL + "/?" + query.Encode()
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		t.Fatal(err)
	}
	for key, values := range headers {
		req.Header[key] = append([]string(nil), values...)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("stream status = %d: %s", resp.StatusCode, body)
	}
	return resp, bufio.NewReader(resp.Body)
}

func readSSEEvent(t *testing.T, reader *bufio.Reader) map[string]string {
	t.Helper()
	event := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read SSE: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if event["id"] != "" || event["event"] != "" || event["data"] != "" {
				return event
			}
			event = make(map[string]string)
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimPrefix(value, " ")
		if key == "data" {
			if event[key] != "" {
				event[key] += "\n"
			}
			event[key] += value
			continue
		}
		event[key] = value
	}
}

func waitUntil(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if condition() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("condition not met within %s", timeout)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type replayFunc func(context.Context, string, string) ([]Event, error)

func (f replayFunc) Replay(ctx context.Context, userID, lastID string) ([]Event, error) {
	return f(ctx, userID, lastID)
}

type recordingBus struct {
	mu        sync.Mutex
	published []BusMessage
	handler   func(BusMessage)
}

func (b *recordingBus) Publish(_ context.Context, message BusMessage) error {
	b.mu.Lock()
	b.published = append(b.published, message)
	b.mu.Unlock()
	return nil
}

func (b *recordingBus) Subscribe(_ context.Context, handler func(BusMessage)) error {
	b.mu.Lock()
	b.handler = handler
	b.mu.Unlock()
	return nil
}

type recordingPresence struct {
	added     atomic.Int64
	removed   atomic.Int64
	refreshed atomic.Int64
	online    bool
}

func (p *recordingPresence) AddConnection(context.Context, string, string) error {
	p.added.Add(1)
	return nil
}

func (p *recordingPresence) RemoveConnection(context.Context, string, string) error {
	p.removed.Add(1)
	return nil
}

func (p *recordingPresence) Refresh(context.Context, string, string) error {
	p.refreshed.Add(1)
	return nil
}

func (p *recordingPresence) Online(context.Context, string) (bool, error) {
	return p.online, nil
}

func assertCloseCode(t *testing.T, event map[string]string, want int) {
	t.Helper()
	if event["event"] != "server_close" {
		t.Fatalf("event = %+v, want server_close", event)
	}
	var payload struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal([]byte(event["data"]), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != want {
		t.Fatalf("close code = %d, want %d", payload.Code, want)
	}
}

type noFlushWriter struct {
	http.ResponseWriter
}

type blockingWriter struct {
	header http.Header
	buffer bytes.Buffer
	gate   chan struct{}
	once   sync.Once
	mu     sync.Mutex
}

func newBlockingWriter() *blockingWriter {
	return &blockingWriter{header: make(http.Header), gate: make(chan struct{})}
}

func (w *blockingWriter) Header() http.Header { return w.header }
func (w *blockingWriter) WriteHeader(int)     {}
func (w *blockingWriter) Flush()              {}

func (w *blockingWriter) Write(data []byte) (int, error) {
	w.once.Do(func() { <-w.gate })
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(data)
}

func (w *blockingWriter) release() { close(w.gate) }

func (w *blockingWriter) data() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]byte(nil), w.buffer.Bytes()...)
}
