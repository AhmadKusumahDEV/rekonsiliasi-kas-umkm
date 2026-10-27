package sse

import "errors"

var (
	ErrAlreadyStarted      = errors.New("sse: manager already started")
	ErrClosed              = errors.New("sse: manager is closed")
	ErrClientClosed        = errors.New("sse: client is closed")
	ErrSlowClient          = errors.New("sse: client send queue is full")
	ErrUserOffline         = errors.New("sse: user is not connected")
	ErrInvalidEvent        = errors.New("sse: invalid event")
	ErrEventTooLarge       = errors.New("sse: event exceeds MaxEventBytes")
	ErrMissingEventID      = errors.New("sse: Event.ID is required when Replay is configured")
	ErrFlusherUnsupported  = errors.New("sse: ResponseWriter does not support flushing")
	ErrConnectionLimit     = errors.New("sse: connection limit reached")
	ErrUserConnectionLimit = errors.New("sse: user connection limit reached")
	ErrRoomLimit           = errors.New("sse: room limit reached")
	ErrNilMiddleware       = errors.New("sse: middleware must not be nil")
)

const (
	CloseServiceRestart = 1012
	CloseSlowClient     = 1013
	CloseTokenExpired   = 4001
	CloseConnectionAge  = 4002
)

// RejectReason describes a stream rejected before the SSE response was opened.
type RejectReason string

const (
	RejectNotReady        RejectReason = "not_ready"
	RejectMethod          RejectReason = "method"
	RejectOrigin          RejectReason = "origin"
	RejectHandshakeLimit  RejectReason = "handshake_limit"
	RejectAuth            RejectReason = "auth"
	RejectIdentity        RejectReason = "identity"
	RejectTokenExpired    RejectReason = "token_expired"
	RejectRoomLimit       RejectReason = "room_limit"
	RejectConnectionLimit RejectReason = "connection_limit"
	RejectUserLimit       RejectReason = "user_connection_limit"
	RejectOnConnect       RejectReason = "on_connect"
	RejectClosed          RejectReason = "closed"
	RejectUnsupported     RejectReason = "streaming_unsupported"
	RejectLastEventID     RejectReason = "last_event_id"
)

// DropReason explains why an event was not queued for a client.
type DropReason string

const (
	DropOutboundQueueFull DropReason = "outbound_queue_full"
)

func validCloseCode(code int) bool {
	return code >= 1000 && code <= 4999 && code != 1004 && code != 1005 && code != 1006 && code != 1015
}
