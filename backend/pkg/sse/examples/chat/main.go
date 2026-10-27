// Command chat is a runnable usage example for pkg/sse.
//
// It shows the full wiring: authentication, manager lifecycle, mounting the
// stream, publishing to a user or room, stats, and graceful shutdown.
//
// Run it with:
//
//	go run ./pkg/sse/examples/chat
//
// Then open http://localhost:8090.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/sse"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	manager, err := sse.New(sse.Config{
		// Demo only: the user and room come from query parameters. In a real
		// service, verify the session or JWT and load room membership here.
		Authenticate:          authenticate,
		HeartbeatInterval:     15 * time.Second,
		SendQueue:             64,
		MaxConnections:        1000,
		MaxConnectionsPerUser: 5,
		Logger:                logger,
		OnConnect: func(ctx context.Context, client *sse.Client) error {
			logger.Info("sse connect", "client", client.ID(), "user", client.UserID(), "rooms", client.Rooms())
			return nil
		},
		OnDisconnect: func(ctx context.Context, client *sse.Client) {
			code, reason := client.CloseInfo()
			logger.Info("sse disconnect", "client", client.ID(), "user", client.UserID(), "code", code, "reason", reason)
		},
		OnDrop: func(ctx context.Context, client *sse.Client, reason sse.DropReason) {
			logger.Warn("sse drop", "client", client.ID(), "user", client.UserID(), "reason", reason)
		},
	})
	if err != nil {
		logger.Error("create sse manager", "err", err)
		os.Exit(1)
	}

	// Use wraps the long-lived stream handler. This middleware does not replace
	// the ResponseWriter, so http.Flusher stays available.
	if err := manager.Use(logStreamLifetime(logger)); err != nil {
		logger.Error("register sse middleware", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := manager.Start(ctx); err != nil {
		logger.Error("start sse manager", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.Handle("/events", manager)
	mux.HandleFunc("/", index)
	mux.HandleFunc("/publish/room", publishToRoom(manager, logger))
	mux.HandleFunc("/publish/user", publishToUser(manager, logger))
	mux.HandleFunc("/stats", showStats(manager))

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8090"
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		// No WriteTimeout: an SSE stream is intentionally long-lived.
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("sse example listening", "addr", addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("signal received, shutting down")
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server", "err", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Close SSE streams before shutting down the HTTP server. Server.Shutdown
	// waits for active handlers, and an SSE handler stays active by design.
	_ = manager.Shutdown(shutdownCtx)
	_ = server.Shutdown(shutdownCtx)
}

func authenticate(r *http.Request) (sse.Identity, error) {
	userID := strings.TrimSpace(r.URL.Query().Get("user"))
	if userID == "" {
		return sse.Identity{}, sse.NewAuthError(http.StatusUnauthorized, "user query parameter is required")
	}
	rooms := make([]string, 0, 2)
	for _, room := range strings.Split(r.URL.Query().Get("room"), ",") {
		if room = strings.TrimSpace(room); room != "" {
			rooms = append(rooms, room)
		}
	}
	return sse.Identity{
		UserID: userID,
		Rooms:  rooms,
		Values: map[string]any{"demo": true},
	}, nil
}

func logStreamLifetime(logger *slog.Logger) sse.Middleware {
	return func(next sse.Handler) sse.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info("sse stream closed", "path", r.URL.Path, "duration", time.Since(start))
		})
	}
}

func publishToRoom(manager *sse.Manager, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		roomID := strings.TrimSpace(r.URL.Query().Get("room"))
		if roomID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "room is required"})
			return
		}
		var body struct {
			Text string `json:"text"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		event, err := sse.NewEvent("message.new", map[string]any{
			"room_id": roomID,
			"text":    body.Text,
		})
		if err != nil {
			logger.Error("build event", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid payload"})
			return
		}

		// Publish* is bus-aware. With Config.Bus set, it reaches every pod.
		// BroadcastToRoom would only reach sockets owned by this process.
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := manager.PublishEventToRoom(ctx, roomID, event); err != nil {
			logger.Error("publish to room", "room", roomID, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "publish failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"published": true, "room": roomID})
	}
}

func publishToUser(manager *sse.Manager, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		userID := strings.TrimSpace(r.URL.Query().Get("user"))
		if userID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user is required"})
			return
		}
		var body struct {
			Text string `json:"text"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		event, err := sse.NewEvent("notice", map[string]any{"text": body.Text})
		if err != nil {
			logger.Error("build event", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid payload"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := manager.PublishEventToUser(ctx, userID, event); err != nil {
			if errors.Is(err, sse.ErrUserOffline) {
				// With a Bus, offline is best-effort: another pod may own the socket.
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "user is offline"})
				return
			}
			logger.Error("publish to user", "user", userID, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "publish failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"published": true, "user": userID})
	}
}

func showStats(manager *sse.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, manager.Stats())
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dest)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>pkg/sse example</title>
<style>
body{font-family:system-ui,sans-serif;max-width:860px;margin:40px auto;padding:0 16px;color:#222}
label{display:block;margin-top:12px;font-weight:600}
input,button{font:inherit;padding:8px;margin-top:4px}
.row{display:flex;gap:8px;flex-wrap:wrap;align-items:end}
#log{background:#111;color:#d7ffd7;padding:12px;height:240px;overflow:auto;white-space:pre-wrap;border-radius:8px;margin-top:16px}
code{background:#f2f2f2;padding:2px 4px;border-radius:4px}
</style>
</head>
<body>
<h1>pkg/sse example</h1>
<p>This page opens a real <code>EventSource</code>. Publish endpoints call <code>Manager.PublishEventToRoom</code> and <code>Manager.PublishEventToUser</code>.</p>
<div class="row">
  <div><label for="user">User</label><input id="user" value="ana"></div>
  <div><label for="room">Room</label><input id="room" value="general"></div>
  <button onclick="connect()">Connect</button>
  <button onclick="disconnect()">Disconnect</button>
</div>
<div class="row">
  <div><label for="text">Message</label><input id="text" value="hello from SSE"></div>
  <button onclick="publishRoom()">Publish to room</button>
  <button onclick="publishUser()">Publish to user</button>
</div>
<div id="log"></div>
<script>
let source = null;
function log(line) {
  const box = document.getElementById("log");
  box.textContent += new Date().toISOString() + " " + line + "\n";
  box.scrollTop = box.scrollHeight;
}
function connect() {
  const user = document.getElementById("user").value.trim();
  const room = document.getElementById("room").value.trim();
  if (!user) { log("user is required"); return; }
  if (source) { source.close(); }
  const url = "/events?user=" + encodeURIComponent(user) + (room ? "&room=" + encodeURIComponent(room) : "");
  log("connecting " + url);
  source = new EventSource(url);
  source.onopen = function () { log("connected"); };
  source.onerror = function () { log("stream error (EventSource retries a broken stream)"); };
  source.addEventListener("message.new", function (event) {
    log("message.new lastEventId=" + event.lastEventId + " data=" + event.data);
  });
  source.addEventListener("notice", function (event) {
    log("notice data=" + event.data);
  });
  source.addEventListener("resync", function (event) {
    log("resync data=" + event.data);
  });
  source.addEventListener("server_close", function (event) {
    log("server_close data=" + event.data);
    source.close();
    source = null;
  });
}
function disconnect() {
  if (source) { source.close(); source = null; log("disconnected"); }
}
async function post(url, body) {
  const response = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body)
  });
  log("POST " + url + " -> " + response.status + " " + await response.text());
}
function publishRoom() {
  const room = document.getElementById("room").value.trim();
  const text = document.getElementById("text").value;
  post("/publish/room?room=" + encodeURIComponent(room), { text: text });
}
function publishUser() {
  const user = document.getElementById("user").value.trim();
  const text = document.getElementById("text").value;
  post("/publish/user?user=" + encodeURIComponent(user), { text: text });
}
</script>
</body>
</html>`
