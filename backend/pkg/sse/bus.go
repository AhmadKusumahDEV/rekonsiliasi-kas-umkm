package sse

import (
	"context"
	"sync"
)

type BusKind string

const (
	BusUser           BusKind = "user"
	BusRoom           BusKind = "room"
	BusRoomExcept     BusKind = "room_except"
	BusAll            BusKind = "all"
	BusJoinUser       BusKind = "join_user"
	BusLeaveUser      BusKind = "leave_user"
	BusDisconnectUser BusKind = "disconnect_user"
	BusCloseRoom      BusKind = "close_room"
)

// BusMessage is one event or control message published through a cross-process
// fanout.
type BusMessage struct {
	Kind         BusKind  `json:"kind"`
	UserID       string   `json:"user_id,omitempty"`
	RoomID       string   `json:"room_id,omitempty"`
	ExceptUserID string   `json:"except_user_id,omitempty"`
	Rooms        []string `json:"rooms,omitempty"`
	Code         int      `json:"code,omitempty"`
	Reason       string   `json:"reason,omitempty"`
	Event        Event    `json:"event"`
}

// Bus is the optional multi-process fanout for events and control messages. A
// publisher must reach every subscriber that should see the message, including
// its own process when that process owns sockets.
type Bus interface {
	Publish(ctx context.Context, message BusMessage) error
	Subscribe(ctx context.Context, handler func(BusMessage)) error
}

type LocalBus struct {
	mu          sync.RWMutex
	subscribers map[*localSubscription]struct{}
}

type localSubscription struct {
	handler func(BusMessage)
}

func NewLocalBus() *LocalBus {
	return &LocalBus{subscribers: make(map[*localSubscription]struct{})}
}

func (b *LocalBus) Publish(_ context.Context, message BusMessage) error {
	b.mu.RLock()
	subs := make([]*localSubscription, 0, len(b.subscribers))
	for sub := range b.subscribers {
		subs = append(subs, sub)
	}
	b.mu.RUnlock()
	for _, sub := range subs {
		sub.handler(message)
	}
	return nil
}

func (b *LocalBus) Subscribe(ctx context.Context, handler func(BusMessage)) error {
	if handler == nil {
		return ErrInvalidEvent
	}
	sub := &localSubscription{handler: handler}
	b.mu.Lock()
	b.subscribers[sub] = struct{}{}
	b.mu.Unlock()
	go func() {
		<-ctx.Done()
		b.mu.Lock()
		delete(b.subscribers, sub)
		b.mu.Unlock()
	}()
	return nil
}
