package echohttp_test

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/echohttp"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/testkit"
	"io"
	"log/slog"
	"net/http"
)

type resetMailer struct {
	to       string
	resetURL string
	calls    int
}

func (m *resetMailer) SendPasswordReset(_ context.Context, to, resetURL string) error {
	m.calls++
	m.to = to
	m.resetURL = resetURL
	return nil
}

func TestPasswordResetFlow(t *testing.T) {
	db := testkit.Prepare(t)
	cfg := config.Config{
		App:    config.AppConfig{Name: "password-reset-test"},
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ShutdownTimeout: time.Second},
		Email:  config.EmailConfig{AppURL: "https://efa.madgo.my.id"},
	}
	svc := service.NewService(cfg, db, nil)
	mailer := &resetMailer{}
	svc.SetPasswordResetMailer(mailer)
	server := &echohttpServer{handler: echohttp.NewEcho(cfg, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))}
	client := &testClient{t: t, server: server}

	email := "reset-" + randomHex() + "@example.test"
	oldPassword := "reset-password-2026"
	newPassword := "reset-password-2027"
	register := client.request(http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Name: "Pemilik Reset", Email: email, Password: oldPassword, Workspace: "Toko Reset",
	}, false)
	requireStatus(t, register, http.StatusCreated)
	client.captureSession(register)

	forgot := client.request(http.MethodPost, "/api/v1/auth/password/forgot", models.ForgotPasswordRequest{Email: email}, false)
	requireStatus(t, forgot, http.StatusAccepted)
	if mailer.calls != 1 || mailer.to != email || mailer.resetURL == "" {
		t.Fatalf("email reset tidak terkirim: %+v", mailer)
	}
	parsed, err := url.Parse(mailer.resetURL)
	if err != nil || !strings.HasPrefix(parsed.Fragment, "reset?token=") {
		t.Fatalf("reset URL tidak valid: %q", mailer.resetURL)
	}
	token := strings.TrimPrefix(parsed.Fragment, "reset?token=")

	reset := client.request(http.MethodPost, "/api/v1/auth/password/reset", models.ResetPasswordRequest{
		Token: token, Password: newPassword,
	}, false)
	requireStatus(t, reset, http.StatusOK)

	me := client.request(http.MethodGet, "/api/v1/auth/me", nil, false)
	requireStatus(t, me, http.StatusUnauthorized)
	oldLogin := client.request(http.MethodPost, "/api/v1/auth/login", models.LoginRequest{Email: email, Password: oldPassword}, false)
	requireStatus(t, oldLogin, http.StatusUnauthorized)
	newLogin := client.request(http.MethodPost, "/api/v1/auth/login", models.LoginRequest{Email: email, Password: newPassword}, false)
	requireStatus(t, newLogin, http.StatusOK)

	reuse := client.request(http.MethodPost, "/api/v1/auth/password/reset", models.ResetPasswordRequest{
		Token: token, Password: newPassword,
	}, false)
	requireStatus(t, reuse, http.StatusBadRequest)

	beforeUnknown := mailer.calls
	unknown := client.request(http.MethodPost, "/api/v1/auth/password/forgot", models.ForgotPasswordRequest{Email: "missing-" + randomHex() + "@example.test"}, false)
	requireStatus(t, unknown, http.StatusAccepted)
	if mailer.calls != beforeUnknown {
		t.Fatal("forgot password mengirim email untuk akun yang tidak ada")
	}
}
