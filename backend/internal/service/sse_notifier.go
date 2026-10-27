package service

import (
	"context"
	"time"
)

// StateNotifier publishes a workspace state version after a committed mutation.
// It is intentionally tiny so the service layer stays independent of Echo and
// the SSE transport package.
type StateNotifier interface {
	PublishState(ctx context.Context, workspaceID string, version int64) error
	DisconnectUser(userID string)
}

func (s *Service) SetStateNotifier(notifier StateNotifier) {
	s.stateNotifier = notifier
}

func (s *Service) publishState(workspaceID string, version int64) {
	if s.stateNotifier == nil || version <= 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.stateNotifier.PublishState(ctx, workspaceID, version)
}

func (s *Service) DisconnectUser(userID string) {
	if s.stateNotifier != nil && userID != "" {
		s.stateNotifier.DisconnectUser(userID)
	}
}
