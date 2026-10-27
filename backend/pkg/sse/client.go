package sse

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type clientOptions struct {
	sendQueue      int
	maxRooms       int
	drainTimeout   time.Duration
	maxEventBytes  int64
	writeTimeout   time.Duration
	eventLimits    map[string]EventLimit
	requireEventID bool
}

// Client is one open SSE stream. A user may have several clients at once.
type Client struct {
	id     string
	userID string
	hub    *Hub
	w      http.ResponseWriter
	rc     *http.ResponseController
	lastID string
	remote string

	send        chan []byte
	done        chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
	closing     atomic.Bool
	gapped      atomic.Bool
	established atomic.Bool
	pending     atomic.Int64
	sendMu      sync.Mutex
	drained     chan struct{}
	drainOnce   sync.Once

	closeOnce   sync.Once
	cleanupOnce sync.Once
	closeMu     sync.Mutex
	closeCode   int
	closeReason string

	roomsMu     sync.RWMutex
	rooms       map[string]struct{}
	values      sync.Map
	connectedAt time.Time

	maxRooms       int
	drainTimeout   time.Duration
	maxEventBytes  int64
	writeTimeout   time.Duration
	eventLimits    map[string]EventLimit
	requireEventID bool
	sequence       atomic.Uint64

	slowMu      sync.Mutex
	slowSince   time.Time
	slowDrops   int
	timerMu     sync.Mutex
	expiryTimer *time.Timer
}

var clientSequence atomic.Uint64

func newClient(ctx context.Context, w http.ResponseWriter, rc *http.ResponseController, r *http.Request, hub *Hub, userID string, options clientOptions) *Client {
	if ctx == nil {
		ctx = context.Background()
	}
	clientCtx, cancel := context.WithCancel(ctx)
	lastID := r.Header.Get("Last-Event-ID")
	if lastID == "" {
		lastID = r.URL.Query().Get("last_event_id")
	}
	return &Client{
		id:             newClientID(),
		userID:         userID,
		hub:            hub,
		w:              w,
		rc:             rc,
		lastID:         lastID,
		remote:         r.RemoteAddr,
		send:           make(chan []byte, options.sendQueue),
		done:           make(chan struct{}),
		ctx:            clientCtx,
		cancel:         cancel,
		rooms:          make(map[string]struct{}),
		connectedAt:    time.Now(),
		maxRooms:       options.maxRooms,
		drainTimeout:   options.drainTimeout,
		maxEventBytes:  options.maxEventBytes,
		writeTimeout:   options.writeTimeout,
		eventLimits:    options.eventLimits,
		drained:        make(chan struct{}),
		requireEventID: options.requireEventID,
	}
}

func (c *Client) ID() string             { return c.id }
func (c *Client) UserID() string         { return c.userID }
func (c *Client) RemoteAddr() string     { return c.remote }
func (c *Client) ConnectedAt() time.Time { return c.connectedAt }
func (c *Client) Done() <-chan struct{}  { return c.done }

// Context is canceled when the stream starts closing. Use it for application
// work tied to this client.
func (c *Client) Context() context.Context { return c.ctx }

// LastEventID is the Last-Event-ID request header (or the last_event_id query
// parameter). Use it with ReplayStore or in OnConnect.
func (c *Client) LastEventID() string { return c.lastID }

func (c *Client) Rooms() []string {
	c.roomsMu.RLock()
	rooms := make([]string, 0, len(c.rooms))
	for roomID := range c.rooms {
		rooms = append(rooms, roomID)
	}
	c.roomsMu.RUnlock()
	sort.Strings(rooms)
	return rooms
}

func (c *Client) Join(roomID string) bool {
	if c.closing.Load() {
		return false
	}
	return c.hub.join(c, roomID, c.maxRooms)
}

func (c *Client) Leave(roomID string) bool {
	if c.closing.Load() {
		return false
	}
	return c.hub.leave(c, roomID)
}

// Send queues one event. With Replay configured, Event.ID must be supplied by
// the application (for example a ULID or store sequence) so Last-Event-ID can
// be resolved later. Without Replay, an empty ID gets a connection-local id.
func (c *Client) Send(event Event) error {
	if limit, ok := c.eventLimits[event.Name]; ok && limit.MaxDataBytes > 0 && int64(len(event.Data)) > limit.MaxDataBytes {
		return ErrEventTooLarge
	}
	if event.ID == "" {
		if c.requireEventID {
			return ErrMissingEventID
		}
		event.ID = fmt.Sprintf("%s-%d", c.id, c.sequence.Add(1))
	}
	frame, err := EncodeEvent(event)
	if err != nil {
		return err
	}
	return c.sendFrame(frame)
}

// SendEvent marshals data and queues it under name.
func (c *Client) SendEvent(name string, data any) error {
	event, err := NewEvent(name, data)
	if err != nil {
		return err
	}
	return c.Send(event)
}

// SendComment queues an SSE comment. It is useful for application-level
// keepalives, though the manager already sends a heartbeat.
func (c *Client) SendComment(text string) error { return c.sendFrame(EncodeComment(text)) }

// Close drains queued events for CloseDrainTimeout and ends the stream.
func (c *Client) Close() error {
	c.gracefulClose(1000, "closed")
	return nil
}

func (c *Client) SetValue(key string, value any) {
	if value == nil {
		c.values.Delete(key)
		return
	}
	c.values.Store(key, value)
}

func (c *Client) Value(key string) (any, bool) { return c.values.Load(key) }

func (c *Client) sendFrame(frame []byte) error {
	if c.maxEventBytes > 0 && int64(len(frame)) > c.maxEventBytes {
		return ErrEventTooLarge
	}
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if c.closing.Load() || c.gapped.Load() {
		return ErrClientClosed
	}
	c.pending.Add(1)
	select {
	case c.send <- frame:
		return nil
	case <-c.done:
		c.completeFrame()
		return ErrClientClosed
	default:
		c.completeFrame()
		if c.hub != nil && c.hub.queueFullPolicy != QueueFullDrop {
			c.gapped.Store(true)
			go c.gracefulClose(CloseSlowClient, "slow client")
		}
		return ErrSlowClient
	}
}

// writeFrame writes and flushes one frame. It is called only by the stream
// goroutine (or during replay before that goroutine starts).
func (c *Client) writeFrame(frame []byte) error {
	if c.rc != nil {
		_ = c.rc.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}
	if _, err := c.w.Write(frame); err != nil {
		return err
	}
	if c.rc != nil {
		return c.rc.Flush()
	}
	return nil
}

func (c *Client) gracefulClose(code int, reason string) {
	c.setCloseInfo(code, reason)
	c.sendMu.Lock()
	if !c.closing.CompareAndSwap(false, true) {
		c.sendMu.Unlock()
		return
	}
	c.sendMu.Unlock()
	c.cancel()

	if c.pending.Load() == 0 {
		c.markDrained()
	}
	timer := time.NewTimer(c.drainTimeout)
	defer timer.Stop()
	select {
	case <-c.drained:
	case <-c.done:
		return
	case <-timer.C:
	}
	c.closeNow(code, reason)
}

func (c *Client) completeFrame() {
	if c.pending.Add(-1) == 0 && c.closing.Load() {
		c.markDrained()
	}
}

func (c *Client) markDrained() {
	c.drainOnce.Do(func() { close(c.drained) })
}

func (c *Client) closeNow(code int, reason string) {
	c.setCloseInfo(code, reason)
	c.sendMu.Lock()
	c.closing.Store(true)
	c.sendMu.Unlock()
	c.closeOnce.Do(func() {
		c.cancel()
		close(c.done)
		c.stopExpiryTimer()
	})
}

func (c *Client) setCloseInfo(code int, reason string) {
	c.closeMu.Lock()
	if c.closeCode == 0 && code != 0 {
		c.closeCode = code
		c.closeReason = reason
	}
	c.closeMu.Unlock()
}

func (c *Client) closeInfo() (int, string) {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if c.closeCode == 0 {
		return 1000, "closed"
	}
	return c.closeCode, c.closeReason
}

// CloseInfo exposes the close code and reason to OnDisconnect and observers.
// It defaults to 1000/"closed" when the client disconnected before a server
// close was requested.
func (c *Client) CloseInfo() (code int, reason string) {
	return c.closeInfo()
}

func (c *Client) setExpiryTimer(timer *time.Timer) {
	c.timerMu.Lock()
	select {
	case <-c.done:
		c.timerMu.Unlock()
		timer.Stop()
		return
	default:
	}
	c.expiryTimer = timer
	c.timerMu.Unlock()
}

func (c *Client) stopExpiryTimer() {
	c.timerMu.Lock()
	if c.expiryTimer != nil {
		c.expiryTimer.Stop()
		c.expiryTimer = nil
	}
	c.timerMu.Unlock()
}

// markGapped stops new frames from entering the queue after a drop. The stream
// is closed next so Last-Event-ID cannot advance past the missing frame.
func (c *Client) markGapped() {
	c.sendMu.Lock()
	c.gapped.Store(true)
	c.sendMu.Unlock()
}

func (c *Client) noteSlow(limit int, window time.Duration) bool {
	if limit <= 0 {
		return false
	}
	c.slowMu.Lock()
	defer c.slowMu.Unlock()
	now := time.Now()
	if c.slowSince.IsZero() || now.Sub(c.slowSince) > window {
		c.slowSince = now
		c.slowDrops = 1
	} else {
		c.slowDrops++
	}
	if c.slowDrops < limit {
		return false
	}
	c.slowDrops = 0
	c.slowSince = time.Time{}
	return true
}

func newClientID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return hex.EncodeToString(raw[:])
	}
	return "s" + strconv.FormatUint(clientSequence.Add(1), 10)
}
