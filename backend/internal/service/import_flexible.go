package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	pkgErr "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/errors"
)

var headerAliases = map[string][]string{
	"id":      {"id", "kode", "source id"},
	"date":    {"tanggal", "tgl", "tanggal transaksi", "tgl trx", "transaction date", "posting date", "date"},
	"ref":     {"referensi", "ref", "no referensi", "no. referensi", "reference", "reference number", "invoice", "no pesanan", "order id", "kode"},
	"name":    {"nama", "nama pelanggan", "pelanggan", "customer", "keterangan", "uraian", "uraian transaksi", "rincian transaksi", "description", "remarks"},
	"amount":  {"nominal", "jumlah", "amount", "total", "nilai", "kredit", "credit", "masuk", "pemasukan"},
	"channel": {"kanal", "channel", "metode", "metode pembayaran", "tipe pembayaran", "cara bayar"},
	"note":    {"catatan", "note", "notes", "keterangan tambahan", "memo"},
}

func duplicateHeaderField(header []string) string {
	seen := make(map[string]int)
	for index, raw := range header {
		name := normalizeHeader(raw)
		for field, aliases := range headerAliases {
			for _, alias := range aliases {
				if name != alias {
					continue
				}
				if previous, exists := seen[field]; exists && previous != index {
					return field
				}
				seen[field] = index
			}
		}
	}
	return ""
}

func isSummaryRow(text string) bool {
	value := normalizePhrase(text)
	for _, word := range []string{"TOTAL", "SUBTOTAL", "JUMLAH", "GRAND TOTAL"} {
		if strings.Contains(" "+value+" ", " "+word+" ") {
			return true
		}
	}
	return false
}

func knownChannel(value string) bool {
	switch value {
	case "tunai", "qris", "transfer", "marketplace", "e-wallet", "lainnya":
		return true
	}
	return false
}

func normalizeHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "\ufeff")))
	return strings.Join(strings.Fields(value), " ")
}

func detectDelimiter(raw string) rune {
	lines := strings.Split(raw, "\n")
	if len(lines) > 10 {
		lines = lines[:10]
	}
	best, bestCount := rune(44), 0
	for _, candidate := range []rune{44, 59, 9} {
		count := 0
		for _, line := range lines {
			count += strings.Count(line, string(candidate))
		}
		if count > bestCount {
			best, bestCount = candidate, count
		}
	}
	return best
}

func resolveColumns(header []string) map[string]int {
	columns := make(map[string]int)
	for index, raw := range header {
		name := normalizeHeader(raw)
		for field, aliases := range headerAliases {
			if _, exists := columns[field]; exists {
				continue
			}
			for _, alias := range aliases {
				if name == alias {
					columns[field] = index
					break
				}
			}
		}
	}
	return columns
}

func parseRupiah(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, errors.New("kosong")
	}
	value = strings.TrimSpace(strings.TrimPrefix(value, "Rp"))
	value = strings.TrimSpace(strings.TrimPrefix(value, "IDR"))
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "\u00a0", "")
	if strings.Count(value, "(") != strings.Count(value, ")") || strings.Contains(value, "(") != strings.Contains(value, ")") {
		return 0, errors.New("kurung tidak seimbang")
	}
	negative := false
	if strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")") {
		negative = true
		value = strings.TrimSpace(value[1 : len(value)-1])
	}
	if strings.HasPrefix(value, "-") {
		negative = !negative
		value = strings.TrimSpace(strings.TrimPrefix(value, "-"))
	}
	if strings.HasPrefix(value, "+") {
		value = strings.TrimSpace(strings.TrimPrefix(value, "+"))
	}
	dotAt := strings.LastIndex(value, ".")
	commaAt := strings.LastIndex(value, ",")
	switch {
	case dotAt >= 0 && commaAt >= 0:
		if dotAt > commaAt {
			value = strings.ReplaceAll(value, ",", "")
		} else {
			value = strings.ReplaceAll(value, ".", "")
			value = strings.ReplaceAll(value, ",", ".")
		}
	case commaAt >= 0:
		parts := strings.Split(value, ",")
		if len(parts) == 2 && len(parts[1]) != 3 {
			value = strings.ReplaceAll(value, ",", ".")
		} else {
			value = strings.ReplaceAll(value, ",", "")
		}
	case dotAt >= 0 && !strings.ContainsAny(value, "eE"):
		parts := strings.Split(value, ".")
		if len(parts) > 2 || (len(parts) == 2 && len(parts[1]) == 3) {
			value = strings.ReplaceAll(value, ".", "")
		}
	}
	amount, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, errors.New("nominal tidak valid")
	}
	if amount > 1e15 || amount < -1e15 {
		return 0, errors.New("nominal terlalu besar")
	}
	result := int64(math.Round(amount))
	if negative {
		result = -result
	}
	return result, nil
}

var dateLayouts = []string{"2006-01-02", "02/01/2006", "2/1/2006", "02-01-2006", "2-1-2006", "02/01/06", "2006/01/02"}

func parseDate(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if idx := strings.IndexAny(value, " T"); idx > 0 {
		value = value[:idx]
	}
	for _, layout := range dateLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format("2006-01-02"), nil
		}
	}
	shown := strings.TrimSpace(raw)
	if len(shown) > 20 {
		shown = shown[:20]
	}
	return "", fmt.Errorf("tanggal %q tidak dikenali", shown)
}

func normalizeChannel(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case value == "":
		return "lainnya"
	case strings.Contains(value, "qris"), value == "qr":
		return "qris"
	case strings.Contains(value, "tunai"), strings.Contains(value, "cash"):
		return "tunai"
	case strings.Contains(value, "transfer"), strings.Contains(value, "trf"), value == "tf", value == "bank transfer":
		return "transfer"
	case strings.Contains(value, "marketplace"), strings.Contains(value, "shopee"), strings.Contains(value, "tokopedia"), strings.Contains(value, "tiktok"), strings.Contains(value, "lazada"):
		return "marketplace"
	case strings.Contains(value, "wallet"), strings.Contains(value, "gopay"), strings.Contains(value, "ovo"), strings.Contains(value, "dana"), strings.Contains(value, "shopeepay"):
		return "e-wallet"
	default:
		return value
	}
}

func autoSourceID(date, ref string, amount int64, row int) string {
	base := fmt.Sprintf("%s|%s|%d|%d", date, normalizeReference(ref), amount, row)
	return "auto-" + shortHash(base)
}

var referencePattern = regexp.MustCompile(`(?i)\b(INV|TRX|REF|INVOICE)[-\s:]?([A-Z0-9]{3,})\b`)

func extractReference(text string) string {
	match := referencePattern.FindStringSubmatch(text)
	if len(match) != 3 {
		return ""
	}
	return strings.ToUpper(match[1]) + "-" + strings.ToUpper(match[2])
}

func shortHash(value string) string {
	hash := uint32(2166136261)
	for i := 0; i < len(value); i++ {
		hash ^= uint32(value[i])
		hash *= 16777619
	}
	return fmt.Sprintf("%08x", hash)
}

// normalizeFlexibleCSV converts an arbitrary header layout into the canonical
// seven columns. It never commits anything; the caller previews first.
func normalizeFlexibleCSV(kind, raw string) (string, []string, error) {
	if kind != "sale" && kind != "receipt" {
		return "", nil, pkgErr.New(pkgErr.ErrInvalidRequest, "Pilih jenis penjualan atau penerimaan.")
	}
	if len(raw) > MaxCSVBytes {
		return "", nil, pkgErr.New(pkgErr.ErrInvalidRequest, "CSV maksimal 1 MB dan 500 baris.")
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(raw, "\ufeff")))
	reader.Comma = detectDelimiter(raw)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return "", nil, pkgErr.New(pkgErr.ErrInvalidRequest, "CSV tidak valid. Periksa tanda kutip dan jumlah kolom.")
	}
	if len(records) == 0 {
		return "", nil, pkgErr.New(pkgErr.ErrInvalidRequest, "CSV kosong atau header tidak valid.")
	}
	headerIndex := -1
	var columns map[string]int
	for index, record := range records {
		if index >= 30 {
			break
		}
		candidate := resolveColumns(record)
		_, hasDate := candidate["date"]
		_, hasAmount := candidate["amount"]
		_, hasName := candidate["name"]
		if hasDate && hasAmount && hasName {
			headerIndex = index
			columns = candidate
			break
		}
	}
	if headerIndex < 0 {
		return "", nil, pkgErr.New(pkgErr.ErrInvalidRequest, "Header tidak ditemukan dalam 30 baris pertama. Gunakan kolom tanggal, nominal, dan nama atau keterangan.")
	}
	if duplicate := duplicateHeaderField(records[headerIndex]); duplicate != "" {
		return "", nil, pkgErr.New(pkgErr.ErrInvalidRequest, "Header duplikat untuk kolom "+duplicate+". Sisakan satu kolom.")
	}

	var builder strings.Builder
	writer := csv.NewWriter(&builder)
	_ = writer.Write([]string{"id", "tanggal", "referensi", "nama", "nominal", "kanal", "catatan"})
	warnings := make([]string, 0)
	rows := 0
	for index, record := range records[headerIndex+1:] {
		line := headerIndex + 2 + index
		rows++
		if len(record) == 0 {
			continue
		}
		get := func(field string) string {
			index, ok := columns[field]
			if !ok || index >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[index])
		}
		name := get("name")
		note := get("note")
		if isSummaryRow(name + " " + note) {
			warnings = append(warnings, fmt.Sprintf("Baris %d adalah baris total dan dilewati.", line))
			continue
		}
		date, dateErr := parseDate(get("date"))
		if dateErr != nil {
			return "", nil, pkgErr.New(pkgErr.ErrInvalidRequest, fmt.Sprintf("Baris %d: %s", line, dateErr.Error()))
		}
		amount, amountErr := parseRupiah(get("amount"))
		if amountErr != nil {
			return "", nil, pkgErr.New(pkgErr.ErrInvalidRequest, fmt.Sprintf("Baris %d: nominal tidak valid.", line))
		}
		if kind == "receipt" && amount <= 0 {
			warnings = append(warnings, fmt.Sprintf("Baris %d dilewati karena bukan uang masuk.", line))
			continue
		}
		if kind == "sale" && amount <= 0 {
			return "", nil, pkgErr.New(pkgErr.ErrInvalidRequest, fmt.Sprintf("Baris %d: nominal penjualan harus lebih dari nol.", line))
		}
		if name == "" {
			name = "Transaksi"
		}
		ref := get("ref")
		if ref == "" {
			ref = extractReference(name + " " + note)
		}
		channel := get("channel")
		if channel == "" {
			inferred := normalizeChannel(name + " " + note)
			if knownChannel(inferred) {
				channel = inferred
			} else {
				channel = "lainnya"
			}
		} else {
			channel = normalizeChannel(channel)
		}
		id := get("id")
		if id == "" {
			id = autoSourceID(date, ref, amount, rows)
		}
		_ = writer.Write([]string{
			id,
			date,
			ref,
			name,
			strconv.FormatInt(amount, 10),
			channel,
			note,
		})
	}
	writer.Flush()
	if err = writer.Error(); err != nil {
		return "", nil, err
	}
	return builder.String(), warnings, nil
}

// FlexibleImport returns a canonical CSV plus the parsed preview for the
// existing import screen. Nothing is stored until the user confirms.
func (s *Service) FlexibleImport(kind, raw string) (models.PreviewResponse, string, []string, error) {
	canonical, warnings, err := normalizeFlexibleCSV(kind, raw)
	if err != nil {
		return models.PreviewResponse{}, "", nil, err
	}
	// A file whose rows were all skipped (for example a receipt export with
	// outgoing transactions only) still returns warnings instead of an error.
	if strings.Count(strings.TrimSpace(canonical), "\n") == 0 {
		return models.PreviewResponse{Rows: []models.Entry{}}, canonical, warnings, nil
	}
	preview, err := s.PreviewImport(kind, canonical)
	if err != nil {
		return models.PreviewResponse{}, "", nil, err
	}
	return preview, canonical, warnings, nil
}

// FlexibleImportForWorkspace also removes rows that already exist in the
// workspace so the preview matches what commit will actually store.
func (s *Service) FlexibleImportForWorkspace(ctx context.Context, user models.User, kind, raw string) (models.PreviewResponse, string, []string, error) {
	canonical, warnings, err := normalizeFlexibleCSV(kind, raw)
	if err != nil {
		return models.PreviewResponse{}, "", nil, err
	}
	if strings.Count(strings.TrimSpace(canonical), "\n") == 0 {
		return models.PreviewResponse{Rows: []models.Entry{}}, canonical, warnings, nil
	}
	preview, err := s.PreviewImportForWorkspace(ctx, user, kind, canonical)
	if err != nil {
		return models.PreviewResponse{}, "", nil, err
	}
	return preview, canonical, warnings, nil
}
