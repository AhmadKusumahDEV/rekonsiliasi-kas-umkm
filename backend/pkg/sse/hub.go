package sse

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Hub indexes local SSE clients by user and room. Membership slices are
// immutable after publication; joins/leaves create a new slice (copy-on-write),
// so a broadcast only reads a slice and does not copy it per event.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	all     []*Client
	users   map[string][]*Client
	rooms   map[string][]*Client

	onDrop          func(context.Context, *Client, DropReason)
	slowLimit       int
	slowWindow      time.Duration
	queueFullPolicy QueueFullPolicy
	maxEventBytes   int64
	eventLimits     map[string]EventLimit
	requireEventID  bool

	eventSeq atomic.Uint64
	sent     atomic.Uint64
}

func newHub() *Hub {
	return &Hub{
		clients:         make(map[*Client]struct{}),
		users:           make(map[string][]*Client),
		rooms:           make(map[string][]*Client),
		queueFullPolicy: QueueFullClose,
	}
}

func (h *Hub) Online(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.users[userID]) > 0
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) UserCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.users)
}

func (h *Hub) UserClientCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.users[userID])
}

func (h *Hub) RoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}

func (h *Hub) Users() []string {
	h.mu.RLock()
	users := make([]string, 0, len(h.users))
	for userID := range h.users {
		users = append(users, userID)
	}
	h.mu.RUnlock()
	sort.Strings(users)
	return users
}

func (h *Hub) Rooms() []string {
	h.mu.RLock()
	rooms := make([]string, 0, len(h.rooms))
	for roomID := range h.rooms {
		rooms = append(rooms, roomID)
	}
	h.mu.RUnlock()
	sort.Strings(rooms)
	return rooms
}

func (h *Hub) RoomClientCount(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[roomID])
}

func (h *Hub) RoomUsers(roomID string) []string {
	h.mu.RLock()
	clients := h.rooms[roomID]
	users := make(map[string]struct{}, len(clients))
	for _, client := range clients {
		users[client.userID] = struct{}{}
	}
	h.mu.RUnlock()
	result := make([]string, 0, len(users))
	for userID := range users {
		result = append(result, userID)
	}
	sort.Strings(result)
	return result
}

func (h *Hub) Clients() []*Client {
	h.mu.RLock()
	clients := append([]*Client(nil), h.all...)
	h.mu.RUnlock()
	return clients
}

func (h *Hub) SentCount() uint64 { return h.sent.Load() }

// SendEventToUser sends one event to every local device of a user.
func (h *Hub) SendEventToUser(userID string, event Event) (int, error) {
	h.mu.RLock()
	clients := h.users[userID]
	h.mu.RUnlock()
	if len(clients) == 0 {
		return 0, ErrUserOffline
	}
	return h.deliver(clients, event, "")
}

func (h *Hub) Broadcast(event Event) (int, error) {
	h.mu.RLock()
	clients := h.all
	h.mu.RUnlock()
	return h.deliver(clients, event, "")
}

func (h *Hub) BroadcastToRoom(roomID string, event Event) (int, error) {
	h.mu.RLock()
	clients := h.rooms[roomID]
	h.mu.RUnlock()
	return h.deliver(clients, event, "")
}

func (h *Hub) BroadcastToRoomExcept(roomID string, event Event, exceptUserID string) (int, error) {
	h.mu.RLock()
	clients := h.rooms[roomID]
	h.mu.RUnlock()
	return h.deliver(clients, event, exceptUserID)
}

func (h *Hub) deliver(clients []*Client, event Event, exceptUserID string) (int, error) {
	if h.requireEventID && event.ID == "" {
		return 0, ErrMissingEventID
	}
	if len(clients) == 0 {
		return 0, nil
	}
	if event.ID == "" {
		event.ID = h.nextEventID()
	}
	if limit, ok := h.eventLimits[event.Name]; ok && limit.MaxDataBytes > 0 && int64(len(event.Data)) > limit.MaxDataBytes {
		return 0, ErrEventTooLarge
	}
	frame, err := EncodeEvent(event)
	if err != nil {
		return 0, err
	}
	if h.maxEventBytes > 0 && int64(len(frame)) > h.maxEventBytes {
		return 0, ErrEventTooLarge
	}

	sent := 0
	var firstErr error
	for _, client := range clients {
		if exceptUserID != "" && client.userID == exceptUserID {
			continue
		}
		if err := client.sendFrame(frame); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			if errors.Is(err, ErrSlowClient) {
				if h.queueFullPolicy != QueueFullDrop {
					client.markGapped()
				}
				h.reportDrop(client, DropOutboundQueueFull)
				if h.queueFullPolicy == QueueFullDrop {
					if client.noteSlow(h.slowLimit, h.slowWindow) {
						go client.gracefulClose(CloseSlowClient, "slow client")
					}
				} else {
					go client.gracefulClose(CloseSlowClient, "slow client")
				}
			}
			continue
		}
		sent++
		h.sent.Add(1)
	}
	if sent == 0 {
		return 0, firstErr
	}
	return sent, nil
}

func (h *Hub) nextEventID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), h.eventSeq.Add(1))
}

func (h *Hub) reportDrop(client *Client, reason DropReason) {
	if h.onDrop != nil {
		h.onDrop(context.Background(), client, reason)
	}
}

func (h *Hub) register(client *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.clients[client]; exists {
		return false
	}
	h.clients[client] = struct{}{}
	h.all = appendClient(h.all, client)
	h.users[client.userID] = appendClient(h.users[client.userID], client)
	return true
}

func (h *Hub) unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.clients[client]; !exists {
		return
	}
	delete(h.clients, client)
	h.all = removeClient(h.all, client)
	if clients := removeClient(h.users[client.userID], client); len(clients) == 0 {
		delete(h.users, client.userID)
	} else {
		h.users[client.userID] = clients
	}

	client.roomsMu.Lock()
	for roomID := range client.rooms {
		if clients := removeClient(h.rooms[roomID], client); len(clients) == 0 {
			delete(h.rooms, roomID)
		} else {
			h.rooms[roomID] = clients
		}
	}
	clear(client.rooms)
	client.roomsMu.Unlock()
}

func (h *Hub) join(client *Client, roomID string, maxRooms int) bool {
	if roomID == "" {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.clients[client]; !exists {
		return false
	}
	client.roomsMu.Lock()
	_, already := client.rooms[roomID]
	roomCount := len(client.rooms)
	client.roomsMu.Unlock()
	if already {
		return true
	}
	if maxRooms > 0 && roomCount >= maxRooms {
		return false
	}
	h.rooms[roomID] = appendClient(h.rooms[roomID], client)
	client.roomsMu.Lock()
	client.rooms[roomID] = struct{}{}
	client.roomsMu.Unlock()
	return true
}

func (h *Hub) leave(client *Client, roomID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.clients[client]; !exists {
		return false
	}
	if clients := removeClient(h.rooms[roomID], client); len(clients) == 0 {
		delete(h.rooms, roomID)
	} else {
		h.rooms[roomID] = clients
	}
	client.roomsMu.Lock()
	delete(client.rooms, roomID)
	client.roomsMu.Unlock()
	return true
}

func (h *Hub) DisconnectUser(userID string, code int, reason string) int {
	h.mu.RLock()
	clients := append([]*Client(nil), h.users[userID]...)
	h.mu.RUnlock()
	var wg sync.WaitGroup
	for _, client := range clients {
		wg.Add(1)
		go func(client *Client) {
			defer wg.Done()
			client.gracefulClose(code, reason)
		}(client)
	}
	wg.Wait()
	return len(clients)
}

// DisconnectUserAsync starts closing every socket of a user and returns
// immediately. Use the blocking DisconnectUser when the caller needs the close
// handshake to finish.
func (h *Hub) DisconnectUserAsync(userID string, code int, reason string) int {
	h.mu.RLock()
	clients := append([]*Client(nil), h.users[userID]...)
	h.mu.RUnlock()
	for _, client := range clients {
		go client.gracefulClose(code, reason)
	}
	return len(clients)
}

func (h *Hub) JoinUser(userID string, roomIDs ...string) int {
	h.mu.RLock()
	clients := append([]*Client(nil), h.users[userID]...)
	h.mu.RUnlock()
	joined := 0
	for _, client := range clients {
		for _, roomID := range roomIDs {
			if client.Join(roomID) {
				joined++
			}
		}
	}
	return joined
}

func (h *Hub) LeaveUser(userID string, roomIDs ...string) int {
	h.mu.RLock()
	clients := append([]*Client(nil), h.users[userID]...)
	h.mu.RUnlock()
	removed := 0
	for _, client := range clients {
		for _, roomID := range roomIDs {
			if client.Leave(roomID) {
				removed++
			}
		}
	}
	return removed
}

func (h *Hub) CloseRoom(roomID string, code int, reason string) int {
	h.mu.RLock()
	clients := append([]*Client(nil), h.rooms[roomID]...)
	h.mu.RUnlock()
	var wg sync.WaitGroup
	for _, client := range clients {
		wg.Add(1)
		go func(client *Client) {
			defer wg.Done()
			client.gracefulClose(code, reason)
		}(client)
	}
	wg.Wait()
	return len(clients)
}

// CloseRoomAsync starts closing every socket in a room and returns immediately.
func (h *Hub) CloseRoomAsync(roomID string, code int, reason string) int {
	h.mu.RLock()
	clients := append([]*Client(nil), h.rooms[roomID]...)
	h.mu.RUnlock()
	for _, client := range clients {
		go client.gracefulClose(code, reason)
	}
	return len(clients)
}

func appendClient(clients []*Client, client *Client) []*Client {
	updated := make([]*Client, len(clients)+1)
	copy(updated, clients)
	updated[len(clients)] = client
	return updated
}

func removeClient(clients []*Client, client *Client) []*Client {
	updated := make([]*Client, 0, len(clients))
	for _, candidate := range clients {
		if candidate != client {
			updated = append(updated, candidate)
		}
	}
	return updated
}
