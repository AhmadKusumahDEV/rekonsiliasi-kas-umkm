package echohttp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	pkgErr "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/errors"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/sse"
)

// NewSSEManager builds the local SSE hub. Authentication is performed at the
// SSE handshake from the existing HttpOnly session cookie; CSRF does not apply
// to GET streams.
func NewSSEManager(cfg config.Config, svc *service.Service, logger *slog.Logger) (*sse.Manager, error) {
	if logger == nil {
		logger = slog.Default()
	}

	return sse.New(sse.Config{
		Authenticate: func(r *http.Request) (sse.Identity, error) {
			cookie, err := r.Cookie(config.SessionCookieName)
			if err != nil || cookie.Value == "" {
				return sse.Identity{}, sse.NewAuthError(http.StatusUnauthorized, "Sesi berakhir. Masuk kembali untuk melanjutkan.")
			}
			session, err := svc.GetSession(r.Context(), cookie.Value)
			if errors.Is(err, pkgErr.ErrUnauthorized) {
				return sse.Identity{}, sse.NewAuthError(http.StatusUnauthorized, "Sesi berakhir. Masuk kembali untuk melanjutkan.")
			}
			if errors.Is(err, pkgErr.ErrServiceUnavailable) {
				return sse.Identity{}, sse.NewAuthError(http.StatusServiceUnavailable, "Penyimpanan sesi belum tersedia.")
			}
			if err != nil {
				return sse.Identity{}, err
			}

			return sse.Identity{
				UserID:    session.User.ID,
				Rooms:     []string{session.User.WorkspaceID},
				ExpiresAt: session.ExpiresAt,
				Values:    map[string]any{"role": session.User.Role},
			}, nil
		},
		AllowedOrigins:        cfg.Server.AllowedOrigins,
		AllowCredentials:      true,
		HeartbeatInterval:     15 * time.Second,
		SendQueue:             64,
		MaxConnections:        10000,
		MaxConnectionsPerUser: 5,
		MaxConnectionAge:      12 * time.Hour,
		Logger:                logger,
		OnReject: func(_ context.Context, r *http.Request, reason sse.RejectReason, err error) {
			logger.Warn("sse rejected", "reason", reason, "path", r.URL.Path, "error", err)
		},
		OnDrop: func(_ context.Context, client *sse.Client, reason sse.DropReason) {
			logger.Warn("sse client dropped", "user_id", client.UserID(), "reason", reason)
		},
	})
}

// SSENotifier bridges committed service mutations to workspace rooms. The
// frontend prototype listens for `state` and compares the numeric workspace
// version before refreshing the REST state.
type SSENotifier struct {
	manager *sse.Manager
	logger  *slog.Logger
}

func NewSSENotifier(manager *sse.Manager, logger *slog.Logger) *SSENotifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &SSENotifier{manager: manager, logger: logger}
}

func (n *SSENotifier) PublishState(ctx context.Context, workspaceID string, version int64) error {
	if n == nil || n.manager == nil {
		return nil
	}
	event, err := sse.NewEvent("state", version)
	if err != nil {
		return fmt.Errorf("build sse state event: %w", err)
	}
	if err := n.manager.PublishEventToRoom(ctx, workspaceID, event); err != nil {
		n.logger.Warn("sse state publish failed", "workspace_id", workspaceID, "version", version, "error", err)
		return err
	}
	return nil
}

func (n *SSENotifier) DisconnectUser(userID string) {
	if n == nil || n.manager == nil || userID == "" {
		return
	}
	n.manager.DisconnectUserAsync(userID, sse.CloseTokenExpired, "session revoked")
}

var _ service.StateNotifier = (*SSENotifier)(nil)
