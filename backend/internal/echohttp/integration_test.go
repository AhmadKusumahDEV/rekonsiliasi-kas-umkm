package echohttp_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/echohttp"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/testkit"
)

type testClient struct {
	t      *testing.T
	server *echohttpServer
	cookie *http.Cookie
	csrf   string
}

type echohttpServer struct {
	handler http.Handler
}

func TestUMKMIntegrationFlow(t *testing.T) {
	db := testkit.Prepare(t)
	cfg := config.Config{
		App:    config.AppConfig{Name: "integration-test"},
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ShutdownTimeout: time.Second, AllowedOrigins: []string{"http://localhost:5173"}},
	}
	svc := service.NewService(cfg, db, nil)
	server := &echohttpServer{handler: echohttp.NewEcho(cfg, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))}
	client := &testClient{t: t, server: server}

	email := "owner+" + randomHex() + "@example.test"
	password := "integration-password-2026"
	register := client.request(http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Name:      "Pemilik Uji",
		Email:     email,
		Password:  password,
		Workspace: "Toko Uji",
	}, false)
	requireStatus(t, register, http.StatusCreated)
	client.captureSession(register)

	me := client.request(http.MethodGet, "/api/v1/auth/me", nil, false)
	requireStatus(t, me, http.StatusOK)
	var auth models.AuthResponse
	decode(t, me, &auth)
	if auth.User.Email != email || auth.CSRF == "" {
		t.Fatalf("unexpected /me response: %+v", auth)
	}

	saleCSV := "id,tanggal,referensi,nama,nominal,kanal,catatan\nS-1,2026-10-01,INV-1,Ayu,100000,transfer,"
	receiptCSV := "id,tanggal,referensi,nama,nominal,kanal,catatan\nR-1,2026-10-02,INV-1,Ayu,100000,transfer,"
	importSale := client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "sale", CSV: saleCSV}, true)
	requireStatus(t, importSale, http.StatusCreated)
	importReceipt := client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "receipt", CSV: receiptCSV}, true)
	requireStatus(t, importReceipt, http.StatusCreated)

	var matches []models.Match
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/suggestions", nil, false), &matches)
	if len(matches) != 1 {
		t.Fatalf("suggestions = %+v, want one match", matches)
	}

	allocationRequest := models.AllocationRequest{
		SaleID:    matches[0].Sale.ID,
		ReceiptID: matches[0].Receipt.ID,
		Amount:    90000,
		Fee:       5000,
		Reason:    "biaya layanan",
		Key:       randomHex(),
	}
	noCSRF := client.request(http.MethodPost, "/api/v1/umkm/allocations", allocationRequest, false)
	requireStatus(t, noCSRF, http.StatusForbidden)
	allocation := client.request(http.MethodPost, "/api/v1/umkm/allocations", allocationRequest, true)
	requireStatus(t, allocation, http.StatusCreated)

	state := client.request(http.MethodGet, "/api/v1/umkm/state", nil, false)
	requireStatus(t, state, http.StatusOK)
	var stateBody models.StateResponse
	decode(t, state, &stateBody)
	if len(stateBody.State.Entries) != 2 || len(stateBody.State.Allocations) != 1 {
		t.Fatalf("unexpected state: %+v", stateBody.State)
	}

	export := client.request(http.MethodGet, "/api/v1/umkm/export", nil, false)
	requireStatus(t, export, http.StatusOK)
	if !bytes.Contains(export.Body.Bytes(), []byte("sisa")) || !bytes.Contains(export.Body.Bytes(), []byte("5000")) {
		t.Fatalf("export does not contain remaining balance: %s", export.Body.String())
	}

	undo := client.request(http.MethodPost, "/api/v1/umkm/allocations/"+allocationRequest.Key+"/undo", nil, true)
	requireStatus(t, undo, http.StatusOK)
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/suggestions", nil, false), &matches)
	if len(matches) != 1 {
		t.Fatalf("suggestions after undo = %+v, want one match", matches)
	}

	var saleImport models.CommitImportResponse
	decode(t, importSale, &saleImport)
	cancel := client.request(http.MethodPost, "/api/v1/umkm/imports/"+saleImport.ID+"/cancel", nil, true)
	requireStatus(t, cancel, http.StatusOK)
	duplicate := client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "sale", CSV: saleCSV}, true)
	requireStatus(t, duplicate, http.StatusConflict)

	logout := client.request(http.MethodPost, "/api/v1/auth/logout", map[string]any{}, true)
	requireStatus(t, logout, http.StatusOK)
	client.cookie = nil
	client.csrf = ""

	login := client.request(http.MethodPost, "/api/v1/auth/login", models.LoginRequest{Email: email, Password: password}, false)
	requireStatus(t, login, http.StatusOK)
	client.captureSession(login)

	change := client.request(http.MethodPut, "/api/v1/auth/password", models.ChangePasswordRequest{
		Current:  password,
		Password: "integration-password-2027",
	}, true)
	requireStatus(t, change, http.StatusOK)

	oldLogin := client.request(http.MethodPost, "/api/v1/auth/login", models.LoginRequest{Email: email, Password: password}, false)
	requireStatus(t, oldLogin, http.StatusUnauthorized)
	newLogin := client.request(http.MethodPost, "/api/v1/auth/login", models.LoginRequest{Email: email, Password: "integration-password-2027"}, false)
	requireStatus(t, newLogin, http.StatusOK)
}

func (c *testClient) request(method, path string, body any, csrf bool) *httptest.ResponseRecorder {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("encode request: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	if csrf {
		req.Header.Set(config.HeaderCSRFToken, c.csrf)
	}
	recorder := httptest.NewRecorder()
	c.server.handler.ServeHTTP(recorder, req)
	return recorder
}

func (c *testClient) captureSession(recorder *httptest.ResponseRecorder) {
	c.t.Helper()
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == config.SessionCookieName {
			c.cookie = cookie
		}
	}
	if c.cookie == nil {
		c.t.Fatalf("session cookie missing; body=%s", recorder.Body.String())
	}
	var response models.AuthResponse
	decode(c.t, recorder, &response)
	if response.CSRF == "" {
		c.t.Fatal("csrf token missing from auth response")
	}
	c.csrf = response.CSRF
}

func requireStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if recorder.Code != want {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, want, recorder.Body.String())
	}
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, recorder.Body.String())
	}
}

func randomHex() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(value[:])
}
