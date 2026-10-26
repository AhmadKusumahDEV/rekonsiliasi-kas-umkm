package echohttp

import (
	"net/http"
	"testing"

	pkgErr "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/errors"
)

func TestDomainErrorMessageAndStatus(t *testing.T) {
	status, message := mapHTTPError(pkgErr.New(pkgErr.ErrConflict, "Data sudah berubah."))
	if status != http.StatusConflict || message != "Data sudah berubah." {
		t.Fatalf("mapping = %d %q, want 409 with public message", status, message)
	}
}
