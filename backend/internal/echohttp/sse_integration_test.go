package echohttp_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/echohttp"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/testkit"
)

func TestSSEManagerPublishesWorkspaceState(t *testing.T) {
	db := testkit.Prepare(t)
	cfg := config.Config{
		App:    config.AppConfig{Name: "sse-integration-test"},
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ShutdownTimeout: time.Second, AllowedOrigins: []string{"http://localhost:5173"}},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.NewService(cfg, db, nil)
	manager, err := echohttp.NewSSEManager(cfg, svc, logger)
	if err != nil {
		t.Fatalf("NewSSEManager: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("manager.Start: %v", err)
	}
	t.Cleanup(func() {
		shutdown, done := context.WithTimeout(context.Background(), 2*time.Second)
		defer done()
		_ = manager.Shutdown(shutdown)
	})

	handler := echohttp.NewEcho(cfg, svc, logger, manager)
	server := httptest.NewServer(handler)
	defer server.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	client := &http.Client{Jar: jar}
	registerBody := bytes.NewBufferString(`{"name":"SSE Test","email":"sse-` + randomHex() + `@example.test","password":"sse-password-2026","workspace":"SSE Workspace"}`)
	registerRequest, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/auth/register", registerBody)
	registerRequest.Header.Set("Content-Type", "application/json")
	registerResponse, err := client.Do(registerRequest)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer registerResponse.Body.Close()
	if registerResponse.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", registerResponse.StatusCode)
	}
	var auth models.AuthResponse
	if err := json.NewDecoder(registerResponse.Body).Decode(&auth); err != nil {
		t.Fatalf("decode register: %v", err)
	}

	streamRequest, _ := http.NewRequest(http.MethodGet, server.URL+"/api/v1/umkm/events", nil)
	streamResponse, err := client.Do(streamRequest)
	if err != nil {
		t.Fatalf("open SSE stream: %v", err)
	}
	defer streamResponse.Body.Close()
	if streamResponse.StatusCode != http.StatusOK {
		t.Fatalf("SSE status = %d", streamResponse.StatusCode)
	}

	deadline := time.Now().Add(2 * time.Second)
	for manager.Stats().ActiveRooms == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if manager.Stats().ActiveRooms == 0 {
		t.Fatal("SSE client never joined its workspace room")
	}

	notifier := echohttp.NewSSENotifier(manager, logger)
	if err := notifier.PublishState(ctx, auth.User.WorkspaceID, 42); err != nil {
		t.Fatalf("publish state: %v", err)
	}

	result := make(chan string, 1)
	go func() {
		reader := bufio.NewReader(streamResponse.Body)
		var eventName, data string
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				result <- ""
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(line, "event: ") {
				eventName = strings.TrimPrefix(line, "event: ")
			}
			if strings.HasPrefix(line, "data: ") {
				data = strings.TrimPrefix(line, "data: ")
			}
			if eventName == "state" && data != "" {
				result <- data
				return
			}
		}
	}()

	select {
	case data := <-result:
		if data != "42" {
			t.Fatalf("SSE state data = %q, want 42", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE state event")
	}
}
