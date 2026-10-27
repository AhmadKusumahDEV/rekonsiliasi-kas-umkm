package sse

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	defaultRetry                   = 3 * time.Second
	defaultHeartbeat               = 15 * time.Second
	defaultWriteTimeout            = 10 * time.Second
	defaultSendQueue               = 256
	defaultCloseDrainTimeout       = 2 * time.Second
	defaultMaxEventBytes           = 1 << 20
	defaultMaxReplayEvents         = 100
	defaultMaxLastEventIDBytes     = 256
	defaultAuthTimeout             = 10 * time.Second
	defaultConnectTimeout          = 10 * time.Second
	defaultCloseTimeout            = 5 * time.Second
	defaultShutdownTimeout         = 10 * time.Second
	defaultMaxConnections          = 10000
	defaultMaxConnectionsPerUser   = 10
	defaultMaxRoomsPerClient       = 1000
	defaultMaxConcurrentHandshakes = 128
	defaultSlowClientLimit         = 3
	defaultSlowClientWindow        = 30 * time.Second
	defaultControlTimeout          = 5 * time.Second
	defaultShutdownCloseCode       = CloseServiceRestart
)

// QueueFullPolicy controls what happens when a client send queue is full.
type QueueFullPolicy string

const (
	// QueueFullClose closes the stream on the first dropped frame. This is the
	// default because a later frame must not advance Last-Event-ID past a gap.
	QueueFullClose QueueFullPolicy = "close"
	// QueueFullDrop keeps the legacy best-effort behavior and can lose events.
	QueueFullDrop QueueFullPolicy = "drop"
)

// Authenticator validates the HTTP request before the SSE stream is opened.
type Authenticator func(*http.Request) (Identity, error)

// Identity is the authenticated user and the rooms joined before streaming.
type Identity struct {
	UserID    string
	Rooms     []string
	Values    map[string]any
	ExpiresAt time.Time
}

// EventLimit applies an optional per-event-type data-size limit. SSE is push
// only, so this is a producer-side size guard, not an inbound rate limit.
type EventLimit struct {
	// MaxDataBytes caps len(Event.Data). Zero disables the type-specific check.
	MaxDataBytes int64
}

// Config controls an SSE Manager. Authenticate is required.
type Config struct {
	Authenticate Authenticator

	// AllowedOrigins lists exact origins or "*". Empty means same host or no
	// Origin header. CheckOrigin replaces it when set.
	AllowedOrigins   []string
	CheckOrigin      func(*http.Request) bool
	AllowCredentials bool

	// Headers are copied onto the SSE response before the stream starts.
	Headers http.Header

	Retry             time.Duration
	HeartbeatInterval time.Duration
	WriteTimeout      time.Duration
	SendQueue         int
	CloseDrainTimeout time.Duration
	MaxEventBytes     int64
	// EventLimits applies optional per-type data-size limits to published events.
	EventLimits map[string]EventLimit

	AuthTimeout           time.Duration
	ConnectTimeout        time.Duration
	CloseTimeout          time.Duration
	ShutdownTimeout       time.Duration
	MaxConnections        int
	MaxConnectionsPerUser int
	MaxRoomsPerClient     int
	// MaxConcurrentHandshakes limits concurrent auth and stream setup, not the
	// number of long-lived streams.
	MaxConcurrentHandshakes int
	MaxConnectionAge        time.Duration

	SlowClientLimit  int
	SlowClientWindow time.Duration
	// QueueFullPolicy controls a full send queue. Empty means QueueFullClose.
	QueueFullPolicy QueueFullPolicy
	// ControlTimeout bounds a Bus control publish for join, leave, disconnect
	// and close-room messages. It applies only when Bus is configured.
	ControlTimeout time.Duration

	ShutdownCloseCode   int
	ShutdownCloseReason string

	OnConnect    func(ctx context.Context, client *Client) error
	OnDisconnect func(ctx context.Context, client *Client)
	OnDrop       func(ctx context.Context, client *Client, reason DropReason)
	// OnPublish observes a server-side publish operation and its duration.
	OnPublish func(ctx context.Context, event Event, duration time.Duration, err error)
	// OnReject runs for a request rejected before the SSE stream opens.
	OnReject func(ctx context.Context, r *http.Request, reason RejectReason, err error)

	// Replay supplies events missed while the client was disconnected. The
	// Last-Event-ID header is passed as lastEventID.
	Replay ReplayStore
	// MaxReplayEvents bounds the replay burst. A truncated replay sends a
	// resync event instead of silently leaving a gap.
	MaxReplayEvents int
	// MaxLastEventIDBytes caps the client-supplied Last-Event-ID header/query.
	MaxLastEventIDBytes int

	// Bus is an optional cross-process fanout. Nil keeps delivery local.
	Bus Bus

	// Presence is optional cross-process online state. Nil uses the local hub.
	// If it also implements PresenceRefresher, heartbeat calls Refresh so TTL
	// entries stay alive while the stream is connected.
	Presence Presence

	Logger *slog.Logger
}

// ReplayStore is implemented by the application when clients must recover
// events after a reconnect. Return events in ascending ID order.
type ReplayStore interface {
	Replay(ctx context.Context, userID, lastEventID string) ([]Event, error)
}

// AuthError lets Authenticate choose the HTTP status before the stream opens.
type AuthError struct {
	Status  int
	Message string
}

func (e *AuthError) Error() string {
	if e == nil {
		return "sse: unauthorized"
	}
	if e.Message != "" {
		return e.Message
	}
	return http.StatusText(e.status())
}

func (e *AuthError) status() int {
	if e == nil || e.Status < 400 || e.Status > 599 {
		return http.StatusUnauthorized
	}
	return e.Status
}

func NewAuthError(status int, message string) *AuthError {
	return &AuthError{Status: status, Message: message}
}

// Validate checks configuration with the same defaults as New.
func (c Config) Validate() error {
	_, err := c.normalize()
	return err
}

func (c Config) normalize() (Config, error) {
	if c.Authenticate == nil {
		return c, errors.New("sse: Config.Authenticate is required")
	}
	if c.Retry == 0 {
		c.Retry = defaultRetry
	}
	if c.HeartbeatInterval == 0 {
		c.HeartbeatInterval = defaultHeartbeat
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = defaultWriteTimeout
	}
	if c.SendQueue == 0 {
		c.SendQueue = defaultSendQueue
	}
	if c.CloseDrainTimeout == 0 {
		c.CloseDrainTimeout = defaultCloseDrainTimeout
	}
	if c.MaxEventBytes == 0 {
		c.MaxEventBytes = defaultMaxEventBytes
	}
	if c.MaxReplayEvents == 0 {
		c.MaxReplayEvents = defaultMaxReplayEvents
	}
	if c.MaxLastEventIDBytes == 0 {
		c.MaxLastEventIDBytes = defaultMaxLastEventIDBytes
	}
	if c.AuthTimeout == 0 {
		c.AuthTimeout = defaultAuthTimeout
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = defaultConnectTimeout
	}
	if c.CloseTimeout == 0 {
		c.CloseTimeout = defaultCloseTimeout
	}
	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = defaultShutdownTimeout
	}
	if c.MaxConnections == 0 {
		c.MaxConnections = defaultMaxConnections
	}
	if c.MaxConnectionsPerUser == 0 {
		c.MaxConnectionsPerUser = defaultMaxConnectionsPerUser
	}
	if c.MaxRoomsPerClient == 0 {
		c.MaxRoomsPerClient = defaultMaxRoomsPerClient
	}
	if c.MaxConcurrentHandshakes == 0 {
		c.MaxConcurrentHandshakes = defaultMaxConcurrentHandshakes
	}
	if c.SlowClientLimit == 0 {
		c.SlowClientLimit = defaultSlowClientLimit
	}
	if c.SlowClientWindow == 0 {
		c.SlowClientWindow = defaultSlowClientWindow
	}
	if c.QueueFullPolicy == "" {
		c.QueueFullPolicy = QueueFullClose
	}
	if c.ControlTimeout == 0 {
		c.ControlTimeout = defaultControlTimeout
	}
	if c.ShutdownCloseCode == 0 {
		c.ShutdownCloseCode = defaultShutdownCloseCode
	}
	if c.ShutdownCloseReason == "" {
		c.ShutdownCloseReason = "service restart; reconnect with jittered backoff"
	}
	if c.Logger == nil {
		c.Logger = slog.New(discardHandler{})
	}

	switch {
	case c.Retry <= 0:
		return c, errors.New("sse: Retry must be positive")
	case c.HeartbeatInterval <= 0:
		return c, errors.New("sse: HeartbeatInterval must be positive")
	case c.WriteTimeout <= 0:
		return c, errors.New("sse: WriteTimeout must be positive")
	case c.SendQueue < 1:
		return c, errors.New("sse: SendQueue must be positive")
	case c.CloseDrainTimeout <= 0:
		return c, errors.New("sse: CloseDrainTimeout must be positive")
	case c.MaxEventBytes < 1:
		return c, errors.New("sse: MaxEventBytes must be positive")
	case c.MaxReplayEvents < 1:
		return c, errors.New("sse: MaxReplayEvents must be positive")
	case c.MaxLastEventIDBytes < 1:
		return c, errors.New("sse: MaxLastEventIDBytes must be positive")
	case c.AuthTimeout <= 0:
		return c, errors.New("sse: AuthTimeout must be positive")
	case c.ConnectTimeout <= 0:
		return c, errors.New("sse: ConnectTimeout must be positive")
	case c.CloseTimeout <= 0:
		return c, errors.New("sse: CloseTimeout must be positive")
	case c.ShutdownTimeout <= 0:
		return c, errors.New("sse: ShutdownTimeout must be positive")
	case c.MaxConnections < -1:
		return c, errors.New("sse: MaxConnections must be -1 or greater")
	case c.MaxConnectionsPerUser < -1:
		return c, errors.New("sse: MaxConnectionsPerUser must be -1 or greater")
	case c.MaxRoomsPerClient < -1:
		return c, errors.New("sse: MaxRoomsPerClient must be -1 or greater")
	case c.MaxConcurrentHandshakes < 1:
		return c, errors.New("sse: MaxConcurrentHandshakes must be positive")
	case c.MaxConnectionAge < 0:
		return c, errors.New("sse: MaxConnectionAge must not be negative")
	case c.SlowClientLimit < -1:
		return c, errors.New("sse: SlowClientLimit must be -1 or greater")
	case c.SlowClientWindow <= 0:
		return c, errors.New("sse: SlowClientWindow must be positive")
	case c.QueueFullPolicy != QueueFullClose && c.QueueFullPolicy != QueueFullDrop:
		return c, errors.New("sse: QueueFullPolicy must be close or drop")
	case c.ControlTimeout <= 0:
		return c, errors.New("sse: ControlTimeout must be positive")
	case !validCloseCode(c.ShutdownCloseCode):
		return c, errors.New("sse: ShutdownCloseCode must be a valid close code")
	}
	for _, origin := range c.AllowedOrigins {
		if strings.TrimSpace(origin) == "" {
			return c, errors.New("sse: AllowedOrigins contains an empty value")
		}
	}
	for eventType, limit := range c.EventLimits {
		if strings.TrimSpace(eventType) == "" {
			return c, errors.New("sse: EventLimits contains an empty event type")
		}
		if limit.MaxDataBytes < 0 {
			return c, errors.New("sse: EventLimits MaxDataBytes must not be negative")
		}
	}
	if c.AllowCredentials && containsOrigin(c.AllowedOrigins, "*") {
		return c, errors.New("sse: AllowedOrigins '*' cannot be combined with AllowCredentials")
	}

	c.AllowedOrigins = append([]string(nil), c.AllowedOrigins...)
	if c.Headers != nil {
		headers := make(http.Header, len(c.Headers))
		for key, values := range c.Headers {
			headers[key] = append([]string(nil), values...)
		}
		c.Headers = headers
	}
	if c.EventLimits != nil {
		limits := make(map[string]EventLimit, len(c.EventLimits))
		for eventType, limit := range c.EventLimits {
			limits[eventType] = limit
		}
		c.EventLimits = limits
	}
	return c, nil
}

// discardHandler is a no-op slog handler for the package default.
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler        { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler             { return discardHandler{} }
