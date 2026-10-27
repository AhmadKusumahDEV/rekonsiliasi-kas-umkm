package sse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// Manager owns the local SSE hub, connection lifecycle and HTTP handler.
type Manager struct {
	cfg    Config
	hub    *Hub
	logger *slog.Logger

	stateMu        sync.Mutex
	started        bool
	closing        bool
	runCtx         context.Context
	runCancel      context.CancelFunc
	startedAt      time.Time
	handshakeSlots chan struct{}

	connWG       sync.WaitGroup
	cleanupWG    sync.WaitGroup
	middlewareMu sync.RWMutex
	middlewares  []Middleware

	totalConnections    atomic.Uint64
	droppedEvents       atomic.Uint64
	rejectedConnections atomic.Uint64
}

// Stats is a point-in-time snapshot.
type Stats struct {
	Started             bool
	Closing             bool
	StartedAt           time.Time
	ConnectedClients    int
	ConnectedUsers      int
	ActiveRooms         int
	TotalConnections    uint64
	EventsSent          uint64
	DroppedEvents       uint64
	RejectedConnections uint64
}

func New(cfg Config) (*Manager, error) {
	normalized, err := cfg.normalize()
	if err != nil {
		return nil, err
	}
	manager := &Manager{
		cfg:            normalized,
		hub:            newHub(),
		logger:         normalized.Logger,
		handshakeSlots: make(chan struct{}, normalized.MaxConcurrentHandshakes),
	}
	manager.hub.onDrop = manager.handleDrop
	manager.hub.slowLimit = normalized.SlowClientLimit
	manager.hub.slowWindow = normalized.SlowClientWindow
	manager.hub.queueFullPolicy = normalized.QueueFullPolicy
	manager.hub.maxEventBytes = normalized.MaxEventBytes
	manager.hub.eventLimits = normalized.EventLimits
	manager.hub.requireEventID = normalized.Replay != nil
	return manager, nil
}

// Start marks the manager ready and subscribes to the optional Bus. Canceling
// ctx triggers a shutdown using Config.ShutdownTimeout.
func (m *Manager) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	m.stateMu.Lock()
	if m.closing {
		m.stateMu.Unlock()
		return ErrClosed
	}
	if m.started {
		m.stateMu.Unlock()
		return ErrAlreadyStarted
	}
	runCtx, cancel := context.WithCancel(ctx)
	m.runCtx = runCtx
	m.runCancel = cancel
	m.started = true
	m.startedAt = time.Now()
	m.stateMu.Unlock()

	if bus := m.cfg.Bus; bus != nil {
		if err := bus.Subscribe(runCtx, func(message BusMessage) {
			if err := m.applyBusMessage(message); err != nil && !errors.Is(err, ErrUserOffline) {
				m.logger.ErrorContext(runCtx, "sse bus message failed", "type", message.Event.Name, "err", err)
			}
		}); err != nil {
			cancel()
			m.stateMu.Lock()
			m.closing = true
			m.stateMu.Unlock()
			return fmt.Errorf("sse: bus subscribe: %w", err)
		}
	}
	go func() {
		<-runCtx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), m.cfg.ShutdownTimeout)
		defer shutdownCancel()
		_ = m.Shutdown(shutdownCtx)
	}()
	return nil
}

// ServeHTTP applies registered middleware around the long-lived SSE handler.
// Mount it on a GET route, for example mux.Handle("/events", manager).
func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var handler http.Handler = http.HandlerFunc(m.serveHTTP)
	m.middlewareMu.RLock()
	middlewares := append([]Middleware(nil), m.middlewares...)
	m.middlewareMu.RUnlock()
	for i := len(middlewares) - 1; i >= 0; i-- {
		if wrapped := middlewares[i](handler); wrapped != nil {
			handler = wrapped
		}
	}
	handler.ServeHTTP(w, r)
}

// Use registers middleware for future SSE HTTP requests. The first middleware
// is outermost. A wrapper must preserve http.Flusher for streaming.
func (m *Manager) Use(middlewares ...Middleware) error {
	for _, middleware := range middlewares {
		if middleware == nil {
			return ErrNilMiddleware
		}
	}
	m.middlewareMu.Lock()
	m.middlewares = append(m.middlewares, middlewares...)
	m.middlewareMu.Unlock()
	return nil
}

func (m *Manager) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		if !checkOrigin(m.cfg, r) {
			m.reject(w, r, http.StatusForbidden, "origin not allowed", RejectOrigin, nil)
			return
		}
		addCORSHeaders(m.cfg, r, w.Header())
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Last-Event-ID")
		w.Header().Set("Access-Control-Max-Age", "600")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		m.reject(w, r, http.StatusMethodNotAllowed, "method not allowed", RejectMethod, nil)
		return
	}
	if !m.ready() {
		m.reject(w, r, http.StatusServiceUnavailable, "sse: server not ready", RejectNotReady, nil)
		return
	}
	if !checkOrigin(m.cfg, r) {
		m.reject(w, r, http.StatusForbidden, "origin not allowed", RejectOrigin, nil)
		return
	}
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID == "" {
		lastEventID = r.URL.Query().Get("last_event_id")
	}
	if len(lastEventID) > m.cfg.MaxLastEventIDBytes {
		m.reject(w, r, http.StatusBadRequest, "last event id is too long", RejectLastEventID, nil)
		return
	}

	select {
	case m.handshakeSlots <- struct{}{}:
		// The slot covers auth and setup, not the lifetime of the stream.
		// It is released explicitly before the long-lived writer loop.
	default:
		m.reject(w, r, http.StatusServiceUnavailable, "sse: too many handshakes", RejectHandshakeLimit, nil)
		return
	}
	holdingSlot := true
	defer func() {
		if holdingSlot {
			<-m.handshakeSlots
		}
	}()

	authCtx, cancelAuth := context.WithTimeout(r.Context(), m.cfg.AuthTimeout)
	identity, err := m.callAuthenticate(r.WithContext(authCtx))
	cancelAuth()
	if err != nil {
		m.writeAuthError(w, r, err)
		return
	}
	if identity.UserID == "" {
		m.reject(w, r, http.StatusUnauthorized, "unauthorized", RejectIdentity, nil)
		return
	}
	if !identity.ExpiresAt.IsZero() && !identity.ExpiresAt.After(time.Now()) {
		m.reject(w, r, http.StatusUnauthorized, "token expired", RejectTokenExpired, nil)
		return
	}
	if m.cfg.MaxRoomsPerClient > 0 && uniqueRoomCount(identity.Rooms) > m.cfg.MaxRoomsPerClient {
		// The room list came from the server-side identity lookup; trimming is
		// safer than rejecting a valid login. The application can log the event.
		m.logger.WarnContext(r.Context(), "sse identity rooms exceed MaxRoomsPerClient; trimming",
			"user_id", identity.UserID, "count", uniqueRoomCount(identity.Rooms), "limit", m.cfg.MaxRoomsPerClient)
		identity.Rooms = trimRooms(identity.Rooms, m.cfg.MaxRoomsPerClient)
	}
	if err := m.checkCapacity(identity.UserID); err != nil {
		m.writeLimitError(w, r, err)
		return
	}

	rc := http.NewResponseController(w)
	client := newClient(m.runCtx, w, rc, r, m.hub, identity.UserID, clientOptions{
		sendQueue:      m.cfg.SendQueue,
		maxRooms:       m.cfg.MaxRoomsPerClient,
		drainTimeout:   m.cfg.CloseDrainTimeout,
		maxEventBytes:  m.cfg.MaxEventBytes,
		writeTimeout:   m.cfg.WriteTimeout,
		eventLimits:    m.cfg.EventLimits,
		requireEventID: m.cfg.Replay != nil,
	})
	if err := m.tryRegister(client); err != nil {
		client.closeNow(1000, "registration rejected")
		m.writeLimitError(w, r, err)
		return
	}
	registered := true
	defer func() {
		if registered {
			m.connWG.Done()
		}
	}()
	streamStarted := false
	defer func() {
		if !streamStarted {
			m.remove(client)
		}
	}()
	m.addPresence(client)

	for _, roomID := range identity.Rooms {
		client.Join(roomID)
	}
	for key, value := range identity.Values {
		client.SetValue(key, value)
	}

	if m.cfg.OnConnect != nil {
		connectCtx, cancel := context.WithTimeout(client.ctx, m.cfg.ConnectTimeout)
		err := m.callOnConnect(connectCtx, client)
		cancel()
		if err != nil {
			m.logger.WarnContext(r.Context(), "sse OnConnect rejected stream", "client_id", client.ID(), "err", err)
			client.closeNow(1008, "connection rejected")
			m.remove(client)
			m.reject(w, r, http.StatusForbidden, "connection rejected", RejectOnConnect, err)
			return
		}
		// OnConnect completed; OnDisconnect must run even if response
		// initialization fails afterward.
		client.established.Store(true)
	}

	if err := m.writeHeaders(w, r); err != nil {
		m.logger.ErrorContext(r.Context(), "sse stream initialization failed", "client_id", client.ID(), "err", err)
		client.closeNow(1000, "stream initialization failed")
		m.remove(client)
		return
	}
	registered = false
	defer m.connWG.Done()
	holdingSlot = false
	<-m.handshakeSlots
	client.established.Store(true)
	m.installExpiry(client, identity)
	m.writeReplay(client)
	streamStarted = true
	m.stream(client, r)
}

func (m *Manager) writeHeaders(w http.ResponseWriter, r *http.Request) error {
	if _, ok := w.(http.Flusher); !ok {
		m.reject(w, r, http.StatusInternalServerError, "streaming unsupported", RejectUnsupported, ErrFlusherUnsupported)
		return ErrFlusherUnsupported
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	if r.ProtoMajor == 1 {
		w.Header().Set("Connection", "keep-alive")
	}
	for key, values := range m.cfg.Headers {
		w.Header()[key] = append([]string(nil), values...)
	}
	addCORSHeaders(m.cfg, r, w.Header())
	w.WriteHeader(http.StatusOK)
	if err := http.NewResponseController(w).Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "retry: %d\n\n", m.cfg.Retry.Milliseconds())
	if err == nil {
		err = http.NewResponseController(w).Flush()
	}
	return err
}

func (m *Manager) writeReplay(client *Client) {
	if m.cfg.Replay == nil || client.lastID == "" {
		return
	}
	events, err := m.callReplay(client)
	if err != nil {
		m.logger.ErrorContext(client.ctx, "sse replay failed", "client_id", client.ID(), "user_id", client.UserID(), "err", err)
		m.writeResync(client, "replay_failed")
		return
	}
	truncated := false
	if len(events) > m.cfg.MaxReplayEvents {
		events = events[:m.cfg.MaxReplayEvents]
		truncated = true
	}
	for _, event := range events {
		if event.ID == "" {
			m.writeResync(client, "replay_missing_id")
			return
		}
		frame, err := EncodeEvent(event)
		if err != nil {
			m.logger.ErrorContext(client.ctx, "sse replay encode failed", "client_id", client.ID(), "err", err)
			m.writeResync(client, "replay_encode_failed")
			return
		}
		if int64(len(frame)) > m.cfg.MaxEventBytes {
			m.writeResync(client, "replay_event_too_large")
			return
		}
		if err := client.writeFrame(frame); err != nil {
			client.closeNow(1000, "replay write failed")
			return
		}
	}
	if truncated {
		m.writeResync(client, "replay_truncated")
	}
}

func (m *Manager) callReplay(client *Client) (events []Event, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			m.logger.ErrorContext(client.ctx, "sse Replay panic", "client_id", client.ID(), "panic", recovered, "stack", string(debug.Stack()))
			err = fmt.Errorf("replay panic: %v", recovered)
		}
	}()
	return m.cfg.Replay.Replay(client.ctx, client.userID, client.lastID)
}

func (m *Manager) writeResync(client *Client, reason string) bool {
	frame, err := EncodeEvent(ResyncEvent(reason))
	if err != nil {
		return false
	}
	if err := client.writeFrame(frame); err != nil {
		client.closeNow(1000, "resync write failed")
		return false
	}
	return true
}

func (m *Manager) stream(client *Client, r *http.Request) {
	heartbeat := time.NewTicker(m.cfg.HeartbeatInterval)
	defer heartbeat.Stop()
	defer m.remove(client)

	for {
		select {
		case <-r.Context().Done():
			client.closeNow(1000, "client disconnected")
			return
		case <-client.done:
			code, reason := client.closeInfo()
			frame, err := EncodeEvent(CloseEvent(code, reason, m.closeRetry(code)))
			if err == nil {
				_ = client.writeFrame(frame)
			}
			return
		case frame := <-client.send:
			err := client.writeFrame(frame)
			client.completeFrame()
			if err != nil {
				client.closeNow(1000, "write failed")
				return
			}
		case <-heartbeat.C:
			go m.refreshPresence(client)
			if err := client.writeFrame(EncodeComment("heartbeat")); err != nil {
				client.closeNow(1000, "heartbeat failed")
				return
			}
		}
	}
}

// Shutdown closes streams with the configured restart code and waits for the
// handlers to finish. It is safe to call more than once.
func (m *Manager) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	m.stateMu.Lock()
	if m.closing {
		m.stateMu.Unlock()
		return m.waitForShutdown(ctx)
	}
	m.closing = true
	if m.runCancel != nil {
		m.runCancel()
	}
	clients := m.hub.Clients()
	started := m.started
	m.stateMu.Unlock()

	var wg sync.WaitGroup
	for _, client := range clients {
		wg.Add(1)
		go func(client *Client) {
			defer wg.Done()
			client.gracefulClose(m.cfg.ShutdownCloseCode, m.cfg.ShutdownCloseReason)
		}(client)
	}
	wg.Wait()
	if !started {
		return nil
	}
	return m.waitForShutdown(ctx)
}

func (m *Manager) Stats() Stats {
	m.stateMu.Lock()
	started := m.started
	closing := m.closing
	startedAt := m.startedAt
	m.stateMu.Unlock()
	return Stats{
		Started:             started,
		Closing:             closing,
		StartedAt:           startedAt,
		ConnectedClients:    m.hub.ClientCount(),
		ConnectedUsers:      m.hub.UserCount(),
		ActiveRooms:         m.hub.RoomCount(),
		TotalConnections:    m.totalConnections.Load(),
		EventsSent:          m.hub.SentCount(),
		DroppedEvents:       m.droppedEvents.Load(),
		RejectedConnections: m.rejectedConnections.Load(),
	}
}

func (m *Manager) Online(userID string) bool { return m.hub.Online(userID) }
func (m *Manager) ConnectedUsers() []string  { return m.hub.Users() }
func (m *Manager) DisconnectUser(userID string, code int, reason string) int {
	count := m.hub.UserClientCount(userID)
	m.publishControl(BusMessage{Kind: BusDisconnectUser, UserID: userID, Code: code, Reason: reason})
	m.hub.DisconnectUser(userID, code, reason)
	return count
}
func (m *Manager) DisconnectUserAsync(userID string, code int, reason string) int {
	count := m.hub.UserClientCount(userID)
	m.publishControlAsync(BusMessage{Kind: BusDisconnectUser, UserID: userID, Code: code, Reason: reason})
	m.hub.DisconnectUserAsync(userID, code, reason)
	return count
}
func (m *Manager) JoinUser(userID string, roomIDs ...string) int {
	joined := m.hub.JoinUser(userID, roomIDs...)
	m.publishControl(BusMessage{Kind: BusJoinUser, UserID: userID, Rooms: roomIDs})
	return joined
}
func (m *Manager) LeaveUser(userID string, roomIDs ...string) int {
	removed := m.hub.LeaveUser(userID, roomIDs...)
	m.publishControl(BusMessage{Kind: BusLeaveUser, UserID: userID, Rooms: roomIDs})
	return removed
}
func (m *Manager) CloseRoom(roomID string, code int, reason string) int {
	count := m.hub.RoomClientCount(roomID)
	m.publishControl(BusMessage{Kind: BusCloseRoom, RoomID: roomID, Code: code, Reason: reason})
	m.hub.CloseRoom(roomID, code, reason)
	return count
}
func (m *Manager) CloseRoomAsync(roomID string, code int, reason string) int {
	count := m.hub.RoomClientCount(roomID)
	m.publishControlAsync(BusMessage{Kind: BusCloseRoom, RoomID: roomID, Code: code, Reason: reason})
	m.hub.CloseRoomAsync(roomID, code, reason)
	return count
}

// publishControl sends a control message when a Bus is configured. Local
// delivery still happens through the hub, so a bus echo may apply it twice;
// join, leave, disconnect and close-room are idempotent.
func (m *Manager) publishControl(message BusMessage) {
	if m.cfg.Bus == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.cfg.ControlTimeout)
	defer cancel()
	if err := m.cfg.Bus.Publish(ctx, message); err != nil {
		m.logger.ErrorContext(ctx, "sse control publish failed", "kind", message.Kind, "err", err)
	}
}

func (m *Manager) publishControlAsync(message BusMessage) {
	if m.cfg.Bus == nil {
		return
	}
	go m.publishControl(message)
}

func (m *Manager) SendEventToUser(userID string, event Event) (int, error) {
	start := time.Now()
	sent, err := m.hub.SendEventToUser(userID, event)
	m.callOnPublish(context.Background(), event, time.Since(start), err)
	return sent, err
}
func (m *Manager) Broadcast(event Event) (int, error) {
	start := time.Now()
	sent, err := m.hub.Broadcast(event)
	m.callOnPublish(context.Background(), event, time.Since(start), err)
	return sent, err
}
func (m *Manager) BroadcastToRoom(roomID string, event Event) (int, error) {
	start := time.Now()
	sent, err := m.hub.BroadcastToRoom(roomID, event)
	m.callOnPublish(context.Background(), event, time.Since(start), err)
	return sent, err
}
func (m *Manager) BroadcastToRoomExcept(roomID string, event Event, exceptUserID string) (int, error) {
	start := time.Now()
	sent, err := m.hub.BroadcastToRoomExcept(roomID, event, exceptUserID)
	m.callOnPublish(context.Background(), event, time.Since(start), err)
	return sent, err
}

func (m *Manager) PublishEventToUser(ctx context.Context, userID string, event Event) error {
	return m.publishEvent(ctx, BusMessage{Kind: BusUser, UserID: userID, Event: event})
}
func (m *Manager) PublishEventToRoom(ctx context.Context, roomID string, event Event) error {
	return m.publishEvent(ctx, BusMessage{Kind: BusRoom, RoomID: roomID, Event: event})
}
func (m *Manager) PublishEventToRoomExcept(ctx context.Context, roomID string, event Event, exceptUserID string) error {
	return m.publishEvent(ctx, BusMessage{Kind: BusRoomExcept, RoomID: roomID, ExceptUserID: exceptUserID, Event: event})
}
func (m *Manager) PublishEventToAll(ctx context.Context, event Event) error {
	return m.publishEvent(ctx, BusMessage{Kind: BusAll, Event: event})
}

func (m *Manager) publishEvent(ctx context.Context, message BusMessage) error {
	start := time.Now()
	if message.Event.ID == "" {
		if m.cfg.Replay != nil {
			err := ErrMissingEventID
			m.callOnPublish(ctx, message.Event, time.Since(start), err)
			return err
		}
		message.Event.ID = m.hub.nextEventID()
	}
	var err error
	if m.cfg.Bus != nil {
		err = m.cfg.Bus.Publish(ctx, message)
	} else {
		err = m.applyBusMessage(message)
	}
	m.callOnPublish(ctx, message.Event, time.Since(start), err)
	return err
}

func (m *Manager) callOnPublish(ctx context.Context, event Event, duration time.Duration, err error) {
	if m.cfg.OnPublish == nil {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			m.logger.ErrorContext(ctx, "sse OnPublish panic", "event", event.Name, "panic", recovered, "stack", string(debug.Stack()))
		}
	}()
	m.cfg.OnPublish(ctx, event, duration, err)
}

func (m *Manager) applyBusMessage(message BusMessage) error {
	switch message.Kind {
	case BusUser:
		_, err := m.hub.SendEventToUser(message.UserID, message.Event)
		return err
	case BusRoom:
		_, err := m.hub.BroadcastToRoom(message.RoomID, message.Event)
		return err
	case BusRoomExcept:
		_, err := m.hub.BroadcastToRoomExcept(message.RoomID, message.Event, message.ExceptUserID)
		return err
	case BusAll:
		_, err := m.hub.Broadcast(message.Event)
		return err
	case BusJoinUser:
		m.hub.JoinUser(message.UserID, message.Rooms...)
		return nil
	case BusLeaveUser:
		m.hub.LeaveUser(message.UserID, message.Rooms...)
		return nil
	case BusDisconnectUser:
		m.hub.DisconnectUser(message.UserID, message.Code, message.Reason)
		return nil
	case BusCloseRoom:
		m.hub.CloseRoom(message.RoomID, message.Code, message.Reason)
		return nil
	default:
		return fmt.Errorf("sse: unknown bus kind %q", message.Kind)
	}
}

func (m *Manager) handleDrop(ctx context.Context, client *Client, reason DropReason) {
	m.droppedEvents.Add(1)
	if m.cfg.OnDrop == nil {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			m.logger.ErrorContext(ctx, "sse OnDrop panic", "client_id", client.ID(), "panic", recovered, "stack", string(debug.Stack()))
		}
	}()
	m.cfg.OnDrop(ctx, client, reason)
}

func (m *Manager) callAuthenticate(r *http.Request) (Identity, error) {
	type result struct {
		identity Identity
		err      error
	}
	results := make(chan result, 1)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				m.logger.ErrorContext(r.Context(), "sse Authenticate panic", "remote", r.RemoteAddr, "panic", recovered, "stack", string(debug.Stack()))
				results <- result{err: fmt.Errorf("authenticate panic: %v", recovered)}
			}
		}()
		identity, err := m.cfg.Authenticate(r)
		results <- result{identity: identity, err: err}
	}()
	select {
	case res := <-results:
		return res.identity, res.err
	case <-r.Context().Done():
		return Identity{}, r.Context().Err()
	}
}

func (m *Manager) callOnConnect(ctx context.Context, client *Client) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			m.logger.ErrorContext(ctx, "sse OnConnect panic", "client_id", client.ID(), "panic", recovered, "stack", string(debug.Stack()))
			err = fmt.Errorf("onconnect panic: %v", recovered)
		}
	}()
	return m.cfg.OnConnect(ctx, client)
}

func (m *Manager) callOnDisconnect(ctx context.Context, client *Client) {
	defer func() {
		if recovered := recover(); recovered != nil {
			m.logger.ErrorContext(ctx, "sse OnDisconnect panic", "client_id", client.ID(), "panic", recovered, "stack", string(debug.Stack()))
		}
	}()
	m.cfg.OnDisconnect(ctx, client)
}

func (m *Manager) ready() bool {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	return m.started && !m.closing
}

func (m *Manager) checkCapacity(userID string) error {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	if m.closing {
		return ErrClosed
	}
	if m.cfg.MaxConnections > 0 && m.hub.ClientCount() >= m.cfg.MaxConnections {
		return ErrConnectionLimit
	}
	if m.cfg.MaxConnectionsPerUser > 0 && m.hub.UserClientCount(userID) >= m.cfg.MaxConnectionsPerUser {
		return ErrUserConnectionLimit
	}
	return nil
}

func (m *Manager) tryRegister(client *Client) error {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	if m.closing {
		return ErrClosed
	}
	if err := m.checkCapacityLocked(client.userID); err != nil {
		return err
	}
	if !m.hub.register(client) {
		return ErrConnectionLimit
	}
	m.connWG.Add(1)
	m.totalConnections.Add(1)
	return nil
}

func (m *Manager) checkCapacityLocked(userID string) error {
	if m.cfg.MaxConnections > 0 && m.hub.ClientCount() >= m.cfg.MaxConnections {
		return ErrConnectionLimit
	}
	if m.cfg.MaxConnectionsPerUser > 0 && m.hub.UserClientCount(userID) >= m.cfg.MaxConnectionsPerUser {
		return ErrUserConnectionLimit
	}
	return nil
}

func (m *Manager) writeLimitError(w http.ResponseWriter, r *http.Request, err error) {
	message := "too many connections"
	status := http.StatusTooManyRequests
	reason := RejectConnectionLimit
	if errors.Is(err, ErrClosed) {
		status = http.StatusServiceUnavailable
		message = "server shutting down"
		reason = RejectClosed
	}
	if errors.Is(err, ErrUserConnectionLimit) {
		message = "too many connections for this user"
		reason = RejectUserLimit
	}
	m.reject(w, r, status, message, reason, err)
}

func (m *Manager) writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	message := "internal error"
	var authErr *AuthError
	switch {
	case errors.As(err, &authErr):
		status = authErr.status()
		if authErr.Message != "" {
			message = authErr.Message
		}
	case errors.Is(err, context.DeadlineExceeded):
		status = http.StatusServiceUnavailable
		message = "authentication timeout"
	}
	if status >= http.StatusInternalServerError {
		m.logger.ErrorContext(r.Context(), "sse authentication failed", "remote", r.RemoteAddr, "status", status, "err", err)
	} else {
		// Do not attach err: token parsers often include the rejected token.
		m.logger.InfoContext(r.Context(), "sse authentication rejected", "remote", r.RemoteAddr, "status", status)
	}
	m.reject(w, r, status, message, RejectAuth, err)
}

func (m *Manager) reject(w http.ResponseWriter, r *http.Request, status int, message string, reason RejectReason, err error) {
	m.rejectedConnections.Add(1)
	if reason != RejectOrigin && reason != RejectMethod {
		addCORSHeaders(m.cfg, r, w.Header())
	}
	if status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable {
		w.Header().Set("Retry-After", "1")
	}
	m.callOnReject(r, reason, err)
	http.Error(w, message, status)
}

func (m *Manager) callOnReject(r *http.Request, reason RejectReason, err error) {
	if m.cfg.OnReject == nil {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			m.logger.ErrorContext(r.Context(), "sse OnReject panic", "reason", reason, "panic", recovered, "stack", string(debug.Stack()))
		}
	}()
	m.cfg.OnReject(r.Context(), r, reason, err)
}

func (m *Manager) installExpiry(client *Client, identity Identity) {
	var closeAt time.Time
	code := 0
	reason := ""
	if !identity.ExpiresAt.IsZero() {
		closeAt = identity.ExpiresAt
		code = CloseTokenExpired
		reason = "token expired"
	}
	if m.cfg.MaxConnectionAge > 0 {
		jitteredAge := time.Duration(float64(m.cfg.MaxConnectionAge) * (0.9 + rand.Float64()*0.2))
		age := client.connectedAt.Add(jitteredAge)
		if closeAt.IsZero() || age.Before(closeAt) {
			closeAt = age
			code = CloseConnectionAge
			reason = "connection age limit"
		}
	}
	if closeAt.IsZero() {
		return
	}
	delay := time.Until(closeAt)
	if delay < 0 {
		delay = 0
	}
	client.setExpiryTimer(time.AfterFunc(delay, func() {
		client.gracefulClose(code, reason)
	}))
}

func (m *Manager) closeRetry(code int) time.Duration {
	switch code {
	case CloseServiceRestart, CloseSlowClient:
		return time.Duration(float64(m.cfg.Retry) * (1 + rand.Float64()))
	case CloseConnectionAge:
		return time.Duration(float64(m.cfg.Retry) * (0.9 + rand.Float64()*0.2))
	default:
		return m.cfg.Retry
	}
}

func (m *Manager) addPresence(client *Client) {
	if m.cfg.Presence == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.cfg.ConnectTimeout)
	defer cancel()
	if err := m.cfg.Presence.AddConnection(ctx, client.userID, client.id); err != nil {
		m.logger.ErrorContext(ctx, "sse presence add failed", "client_id", client.ID(), "err", err)
	}
}

func (m *Manager) removePresence(client *Client) {
	if m.cfg.Presence == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.cfg.CloseTimeout)
	defer cancel()
	if err := m.cfg.Presence.RemoveConnection(ctx, client.userID, client.id); err != nil {
		m.logger.ErrorContext(ctx, "sse presence remove failed", "client_id", client.ID(), "err", err)
	}
}

func (m *Manager) refreshPresence(client *Client) {
	if m.cfg.Presence == nil {
		return
	}
	refresher, ok := m.cfg.Presence.(PresenceRefresher)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.cfg.ConnectTimeout)
	defer cancel()
	if err := refresher.Refresh(ctx, client.userID, client.id); err != nil {
		m.logger.ErrorContext(ctx, "sse presence refresh failed", "client_id", client.ID(), "err", err)
	}
}

func (m *Manager) remove(client *Client) {
	client.cleanupOnce.Do(func() {
		m.hub.unregister(client)
		client.closeNow(1000, "closed")
		m.cleanupWG.Add(1)
		go func() {
			defer m.cleanupWG.Done()
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				m.removePresence(client)
			}()
			if client.established.Load() && m.cfg.OnDisconnect != nil {
				wg.Add(1)
				go func() {
					defer wg.Done()
					ctx, cancel := context.WithTimeout(context.Background(), m.cfg.CloseTimeout)
					defer cancel()
					m.callOnDisconnect(ctx, client)
				}()
			}
			wg.Wait()
		}()
	})
}

func (m *Manager) waitForShutdown(ctx context.Context) error {
	if err := waitForGroup(ctx, &m.connWG); err != nil {
		return err
	}
	return waitForGroup(ctx, &m.cleanupWG)
}

func waitForGroup(ctx context.Context, group *sync.WaitGroup) error {
	done := make(chan struct{})
	go func() {
		group.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func uniqueRoomCount(rooms []string) int {
	seen := make(map[string]struct{}, len(rooms))
	for _, roomID := range rooms {
		if roomID != "" {
			seen[roomID] = struct{}{}
		}
	}
	return len(seen)
}

func trimRooms(rooms []string, limit int) []string {
	if limit < 0 {
		return rooms
	}
	if limit == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(rooms))
	trimmed := make([]string, 0, min(len(rooms), limit))
	for _, roomID := range rooms {
		if roomID == "" {
			continue
		}
		if _, exists := seen[roomID]; exists {
			continue
		}
		seen[roomID] = struct{}{}
		trimmed = append(trimmed, roomID)
		if len(trimmed) == limit {
			break
		}
	}
	return trimmed
}
