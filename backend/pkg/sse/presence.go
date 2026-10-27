package sse

import "context"

// Presence optionally tracks online users across processes. The local hub is
// always checked first.
type Presence interface {
	AddConnection(ctx context.Context, userID, connectionID string) error
	RemoveConnection(ctx context.Context, userID, connectionID string) error
	Online(ctx context.Context, userID string) (bool, error)
}

func (m *Manager) IsOnline(ctx context.Context, userID string) (bool, error) {
	if m.hub.Online(userID) {
		return true, nil
	}
	if m.cfg.Presence == nil {
		return false, nil
	}
	return m.cfg.Presence.Online(ctx, userID)
}
