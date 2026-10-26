// Package errors holds sentinel errors and the messages returned by the HTTP
// error handler. Add a new sentinel here, then map it in
// internal/echohttp/echohttp_error_handler.go.
package errors

import "errors"

// DomainError carries a safe, user-facing message while still matching a
// sentinel with errors.Is. Internal SQL/driver details never go in Message.
type DomainError struct {
	Kind    error
	Message string
}

func (e *DomainError) Error() string { return e.Message }

func (e *DomainError) Unwrap() error { return e.Kind }

func New(kind error, message string) error {
	return &DomainError{Kind: kind, Message: message}
}

var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrFailInitializeTx   = errors.New("failed to initialize transaction")
	ErrFailCommitTx       = errors.New("failed to commit transaction")
)

var (
	MsgInvalidRequest     = "Data permintaan tidak valid."
	MsgUnauthorized       = "Sesi tidak valid atau sudah berakhir."
	MsgForbidden          = "Kamu tidak memiliki akses ke aksi ini."
	MsgNotFound           = "Data yang diminta tidak ditemukan."
	MsgConflict           = "Data bertentangan dengan kondisi saat ini."
	MsgServiceUnavailable = "Layanan sedang tidak tersedia."
	MsgInternal           = "Terjadi kesalahan pada server."
)
