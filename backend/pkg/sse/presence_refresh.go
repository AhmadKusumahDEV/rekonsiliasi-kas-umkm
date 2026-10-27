package sse

import "context"

// PresenceRefresher is an optional extension to Presence. The manager calls
// Refresh on each heartbeat so a TTL-backed presence entry stays alive while
// the stream is connected.
type PresenceRefresher interface {
	Refresh(ctx context.Context, userID, connectionID string) error
}
