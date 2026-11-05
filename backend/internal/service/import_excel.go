package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	pkgErr "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/errors"
	"github.com/xuri/excelize/v2"
)

const maxUploadBytes = 10 << 20

type SheetRequiredError struct {
	Sheets []string
}

func (e *SheetRequiredError) Error() string {
	return "pilih sheet yang ingin diimpor"
}

func xlsxEnabled() bool {
	value := strings.TrimSpace(os.Getenv("IMPORT_XLSX_ENABLED"))
	switch strings.ToLower(value) {
	case "false", "0", "off", "no":
		return false
	}
	return true
}

func detectHeaderRow(rows [][]string) int {
	for index, row := range rows {
		if index >= 30 {
			break
		}
		columns := resolveColumns(row)
		_, hasDate := columns["date"]
		_, hasAmount := columns["amount"]
		_, hasName := columns["name"]
		if hasDate && hasAmount && hasName {
			return index + 1
		}
	}
	return 0
}

func rowsToCSV(rows [][]string) (string, error) {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return "", err
		}
	}
	writer.Flush()
	return builder.String(), writer.Error()
}

func (s *Service) ImportUploadedFile(ctx context.Context, user models.User, kind, sheet, filename string, data []byte) (models.FlexibleImportResponse, error) {
	if kind != "sale" && kind != "receipt" {
		return models.FlexibleImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "Pilih jenis penjualan atau penerimaan.")
	}
	if len(data) == 0 {
		return models.FlexibleImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "File kosong.")
	}
	if len(data) > maxUploadBytes {
		return models.FlexibleImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "File maksimal 10 MB.")
	}
	if bytes.HasPrefix(data, []byte{0xD0, 0xCF, 0x11, 0xE0}) {
		return models.FlexibleImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "Format .xls lama belum didukung. Simpan sebagai .xlsx atau CSV.")
	}
	if bytes.HasPrefix(data, []byte("%PDF")) {
		return models.FlexibleImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "PDF belum didukung. Ekspor sebagai CSV atau XLSX.")
	}
	if !bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		preview, canonical, warnings, err := s.FlexibleImportForWorkspace(ctx, user, kind, string(data))
		if err != nil {
			return models.FlexibleImportResponse{}, err
		}
		response := buildUploadResponse(preview, canonical, warnings)
		response.Source = &models.ImportSource{Format: "csv"}
		return response, nil
	}
	if !xlsxEnabled() {
		return models.FlexibleImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "Unggah XLSX belum diaktifkan.")
	}

	workbook, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{
		UnzipSizeLimit:    100 << 20,
		UnzipXMLSizeLimit: 16 << 20,
		RawCellValue:      true,
	})
	if err != nil {
		return models.FlexibleImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "File Excel tidak bisa dibaca. Periksa apakah file terenkripsi atau rusak.")
	}
	defer func() { _ = workbook.Close() }()

	sheets := workbook.GetSheetList()
	if len(sheets) == 0 {
		return models.FlexibleImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "Workbook tidak memiliki sheet yang bisa dibaca.")
	}
	if len(sheets) > 20 {
		return models.FlexibleImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "Workbook maksimal 20 sheet.")
	}
	chosen := strings.TrimSpace(sheet)
	if chosen != "" {
		found := false
		for _, name := range sheets {
			if name == chosen {
				found = true
				break
			}
		}
		if !found {
			return models.FlexibleImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "Sheet tidak ditemukan: "+chosen)
		}
	}
	if chosen == "" {
		candidates := make([]string, 0)
		for _, name := range sheets {
			visible, visErr := workbook.GetSheetVisible(name)
			if visErr != nil || !visible {
				continue
			}
			rows, rowsErr := workbook.GetRows(name, excelize.Options{RawCellValue: true})
			if rowsErr == nil && len(rows) > 0 {
				candidates = append(candidates, name)
			}
		}
		if len(candidates) == 0 {
			return models.FlexibleImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "Tidak ada sheet berisi data.")
		}
		if len(candidates) > 1 {
			return models.FlexibleImportResponse{}, &SheetRequiredError{Sheets: candidates}
		}
		chosen = candidates[0]
	}
	rows, err := workbook.GetRows(chosen, excelize.Options{RawCellValue: true})
	if err != nil {
		return models.FlexibleImportResponse{}, fmt.Errorf("read sheet: %w", err)
	}
	if len(rows) == 0 {
		return models.FlexibleImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "Sheet kosong.")
	}
	if dimension, dimErr := workbook.GetSheetDimension(chosen); dimErr == nil {
		if dimRows := dimensionRows(dimension); dimRows > len(rows)+1 {
			return models.FlexibleImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "Sheet tidak lengkap atau ada baris yang gagal dibaca.")
		}
	}
	if len(rows) > MaxImportRows+30 {
		return models.FlexibleImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "Sheet melebihi batas 500 baris data.")
	}
	headerRow := detectHeaderRow(rows)
	props, _ := workbook.GetWorkbookProps()
	date1904 := props.Date1904 != nil && *props.Date1904
	normalizeExcelDates(workbook, chosen, rows, date1904)
	raw, err := rowsToCSV(rows)
	if err != nil {
		return models.FlexibleImportResponse{}, fmt.Errorf("convert sheet: %w", err)
	}
	preview, canonical, warnings, err := s.FlexibleImportForWorkspace(ctx, user, kind, raw)
	if err != nil {
		return models.FlexibleImportResponse{}, err
	}
	response := buildUploadResponse(preview, canonical, warnings)
	response.Source = &models.ImportSource{Format: "xlsx", Sheet: chosen, HeaderRow: headerRow, Sheets: sheets}
	return response, nil
}

func buildUploadResponse(preview models.PreviewResponse, canonical string, warnings []string) models.FlexibleImportResponse {
	if warnings == nil {
		warnings = []string{}
	}
	return models.FlexibleImportResponse{
		CSV:      canonical,
		Count:    preview.Count,
		Total:    preview.Total,
		Checksum: preview.Checksum,
		Preview:  preview.Rows,
		Skipped:  preview.Skipped,
		Warnings: warnings,
	}
}

func AsSheetRequired(err error) (*SheetRequiredError, bool) {
	var sheetErr *SheetRequiredError
	if errors.As(err, &sheetErr) {
		return sheetErr, true
	}
	return nil, false
}
