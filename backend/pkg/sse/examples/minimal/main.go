// Command minimal is the smallest useful pkg/sse server.
//
//	go run ./pkg/sse/examples/minimal
//
// Terminal 1:
//
//	curl -N -H "X-User: ana" http://localhost:8090/events
//
// Terminal 2:
//
//	curl -X POST "http://localhost:8090/notify?user=ana"
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/sse"
)

func main() {
	manager, err := sse.New(sse.Config{
		Authenticate: func(r *http.Request) (sse.Identity, error) {
			userID := strings.TrimSpace(r.Header.Get("X-User"))
			if userID == "" {
				return sse.Identity{}, sse.NewAuthError(http.StatusUnauthorized, "X-User is required")
			}
			return sse.Identity{UserID: userID}, nil
		},
		HeartbeatInterval: 15 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := manager.Start(ctx); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/events", manager)
	mux.HandleFunc("/notify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		userID := strings.TrimSpace(r.URL.Query().Get("user"))
		if userID == "" {
			http.Error(w, "user is required", http.StatusBadRequest)
			return
		}
		event, err := sse.NewEvent("notice", map[string]string{"text": "hello"})
		if err != nil {
			http.Error(w, "invalid event", http.StatusInternalServerError)
			return
		}
		publishCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := manager.PublishEventToUser(publishCtx, userID, event); err != nil {
			if errors.Is(err, sse.ErrUserOffline) {
				http.Error(w, "user is offline", http.StatusNotFound)
				return
			}
			http.Error(w, "publish failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8090"
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		// No WriteTimeout: SSE streams are long-lived.
	}
	go func() {
		log.Printf("listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = manager.Shutdown(shutdownCtx)
	_ = server.Shutdown(shutdownCtx)
}
