package echohttp_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/echohttp"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/testkit"
	"github.com/xuri/excelize/v2"
)

func xlsxBytes(t *testing.T, sheets map[string][][]string) []byte {
	t.Helper()
	f := excelize.NewFile()
	first := true
	for name, rows := range sheets {
		if first {
			if err := f.SetSheetName("Sheet1", name); err != nil {
				t.Fatal(err)
			}
			first = false
		} else if _, err := f.NewSheet(name); err != nil {
			t.Fatal(err)
		}
		for i, row := range rows {
			cell, _ := excelize.CoordinatesToCellName(1, i+1)
			if err := f.SetSheetRow(name, cell, &row); err != nil {
				t.Fatal(err)
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestUploadXLSXSerialDate(t *testing.T) {
	db := testkit.Prepare(t)
	cfg := config.Config{App: config.AppConfig{Name: "xlsx-date"}, Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ShutdownTimeout: time.Second}}
	svc := service.NewService(cfg, db, nil)
	server := &echohttpServer{handler: echohttp.NewEcho(cfg, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))}
	client := &testClient{t: t, server: server}
	register := client.request(http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Name: "Date", Email: "date-" + randomHex() + "@example.test", Password: "date-password-2026", Workspace: "Toko Date",
	}, false)
	requireStatus(t, register, http.StatusCreated)
	client.captureSession(register)

	f := excelize.NewFile()
	_ = f.SetSheetRow("Sheet1", "A1", &[]string{"Tanggal", "Nama", "Nominal"})
	_ = f.SetCellValue("Sheet1", "A2", time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC))
	_ = f.SetCellValue("Sheet1", "B2", "Budi")
	_ = f.SetCellValue("Sheet1", "C2", 100000)
	style, err := f.NewStyle(&excelize.Style{NumFmt: 14})
	if err != nil {
		t.Fatal(err)
	}
	_ = f.SetCellStyle("Sheet1", "A2", "A2", style)
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	rec := client.upload(t, "sale", buf.Bytes(), "tanggal.xlsx")
	requireStatus(t, rec, http.StatusOK)
	var preview models.FlexibleImportResponse
	decode(t, rec, &preview)
	if !strings.Contains(preview.CSV, "2026-11-03") {
		t.Fatalf("csv = %s", preview.CSV)
	}
}

func TestUploadXLSXSheetSelectionAndLargeSheet(t *testing.T) {
	t.Setenv("IMPORT_XLSX_ENABLED", "true")
	db := testkit.Prepare(t)
	cfg := config.Config{App: config.AppConfig{Name: "xlsx-sheet-select"}, Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ShutdownTimeout: time.Second}}
	svc := service.NewService(cfg, db, nil)
	server := &echohttpServer{handler: echohttp.NewEcho(cfg, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))}
	client := &testClient{t: t, server: server}
	register := client.request(http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Name: "Select", Email: "select-" + randomHex() + "@example.test", Password: "select-password-2026", Workspace: "Toko Select",
	}, false)
	requireStatus(t, register, http.StatusCreated)
	client.captureSession(register)

	data := xlsxBytes(t, map[string][][]string{
		"Penjualan": {{"Tanggal", "Nama", "Nominal"}, {"2026-11-03", "Budi", "100000"}},
		"Rekap":     {{"Tanggal", "Nama", "Nominal"}, {"2026-11-04", "Sari", "50000"}},
	})
	rec := client.uploadWithSheet(t, "sale", data, "pilih.xlsx", "Rekap")
	requireStatus(t, rec, http.StatusOK)
	var chosen models.FlexibleImportResponse
	decode(t, rec, &chosen)
	if chosen.Source == nil || chosen.Source.Sheet != "Rekap" || chosen.Count != 1 {
		t.Fatalf("chosen = %+v", chosen)
	}

	rows := make([][]string, 0, 98)
	for i := 0; i < 7; i++ {
		rows = append(rows, []string{"Rekap penjualan"})
	}
	rows = append(rows, []string{"Tanggal", "Nama", "Nominal", "Kanal"})
	for i := 0; i < 90; i++ {
		rows = append(rows, []string{"2026-11-03", "Pelanggan " + strconv.Itoa(i), "1000", "transfer"})
	}
	large := xlsxBytes(t, map[string][][]string{"Data": rows})
	rec = client.upload(t, "sale", large, "besar.xlsx")
	requireStatus(t, rec, http.StatusOK)
	var big models.FlexibleImportResponse
	decode(t, rec, &big)
	if big.Count != 90 || big.Source == nil || big.Source.HeaderRow != 8 {
		t.Fatalf("big count=%d source=%+v", big.Count, big.Source)
	}
}

func (c *testClient) upload(t *testing.T, kind string, data []byte, filename string) *httptest.ResponseRecorder {
	return c.uploadWithSheet(t, kind, data, filename, "")
}

func (c *testClient) uploadWithSheet(t *testing.T, kind string, data []byte, filename, sheet string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = w.WriteField("kind", kind)
	if sheet != "" {
		_ = w.WriteField("sheet", sheet)
	}
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/umkm/imports/files", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	if c.csrf != "" {
		req.Header.Set("X-CSRF-Token", c.csrf)
	}
	rec := httptest.NewRecorder()
	c.server.handler.ServeHTTP(rec, req)
	return rec
}

func TestUploadXLSXAndCSV(t *testing.T) {
	t.Setenv("IMPORT_XLSX_ENABLED", "true")
	db := testkit.Prepare(t)
	cfg := config.Config{App: config.AppConfig{Name: "xlsx-test"}, Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ShutdownTimeout: time.Second}}
	svc := service.NewService(cfg, db, nil)
	server := &echohttpServer{handler: echohttp.NewEcho(cfg, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))}
	client := &testClient{t: t, server: server}
	register := client.request(http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Name: "XLSX", Email: "xlsx-" + randomHex() + "@example.test", Password: "xlsx-password-2026", Workspace: "Toko XLSX",
	}, false)
	requireStatus(t, register, http.StatusCreated)
	client.captureSession(register)

	data := xlsxBytes(t, map[string][][]string{
		"Penjualan": {
			{"Rekap Penjualan"},
			{"Tanggal", "Nama", "Nominal", "Kanal", "Referensi"},
			{"2026-11-03", "Budi", "100000", "transfer", "INV-1"},
			{"2026-11-03", "Sari", "50000", "qris", "INV-2"},
		},
	})
	rec := client.upload(t, "sale", data, "penjualan.xlsx")
	requireStatus(t, rec, http.StatusOK)
	var preview models.FlexibleImportResponse
	decode(t, rec, &preview)
	if preview.Source == nil || preview.Source.Format != "xlsx" || preview.Source.Sheet != "Penjualan" {
		t.Fatalf("source = %+v", preview.Source)
	}
	if preview.Count != 2 || !strings.Contains(preview.CSV, "Budi") {
		t.Fatalf("preview count=%d csv=%s", preview.Count, preview.CSV)
	}
	commit := client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "sale", CSV: preview.CSV}, true)
	requireStatus(t, commit, http.StatusCreated)

	csvData := []byte("tanggal,nama,nominal,kanal\n2026-11-04,Dina,75000,transfer\n")
	rec = client.upload(t, "sale", csvData, "tambahan.csv")
	requireStatus(t, rec, http.StatusOK)
	var csvPreview models.FlexibleImportResponse
	decode(t, rec, &csvPreview)
	if csvPreview.Source == nil || csvPreview.Source.Format != "csv" || csvPreview.Count != 1 {
		t.Fatalf("csv preview = %+v", csvPreview)
	}
}

func TestUploadXLSXRequiresSheetChoice(t *testing.T) {
	t.Setenv("IMPORT_XLSX_ENABLED", "true")
	db := testkit.Prepare(t)
	cfg := config.Config{App: config.AppConfig{Name: "xlsx-sheet"}, Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ShutdownTimeout: time.Second}}
	svc := service.NewService(cfg, db, nil)
	server := &echohttpServer{handler: echohttp.NewEcho(cfg, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))}
	client := &testClient{t: t, server: server}
	register := client.request(http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Name: "Sheet", Email: "sheet-" + randomHex() + "@example.test", Password: "sheet-password-2026", Workspace: "Toko Sheet",
	}, false)
	requireStatus(t, register, http.StatusCreated)
	client.captureSession(register)

	data := xlsxBytes(t, map[string][][]string{
		"Penjualan": {{"Tanggal", "Nama", "Nominal"}, {"2026-11-03", "Budi", "100000"}},
		"Rekap":     {{"Tanggal", "Nama", "Nominal"}, {"2026-11-04", "Sari", "50000"}},
	})
	rec := client.upload(t, "sale", data, "dua-sheet.xlsx")
	requireStatus(t, rec, http.StatusUnprocessableEntity)
	var body struct {
		Details struct {
			Sheets []string `json:"sheets"`
		} `json:"details"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Details.Sheets) != 2 {
		t.Fatalf("sheets = %v", body.Details.Sheets)
	}
}

func TestUploadRejectsCorruptXLSX(t *testing.T) {
	db := testkit.Prepare(t)
	cfg := config.Config{App: config.AppConfig{Name: "xlsx-bad"}, Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ShutdownTimeout: time.Second}}
	svc := service.NewService(cfg, db, nil)
	server := &echohttpServer{handler: echohttp.NewEcho(cfg, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))}
	client := &testClient{t: t, server: server}
	register := client.request(http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Name: "Bad", Email: "bad-" + randomHex() + "@example.test", Password: "bad-password-2026", Workspace: "Toko Bad",
	}, false)
	requireStatus(t, register, http.StatusCreated)
	client.captureSession(register)
	rec := client.upload(t, "sale", []byte("PK\x03\x04garbage"), "rusak.xlsx")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
}
