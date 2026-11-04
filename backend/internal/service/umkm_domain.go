package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	pkgErr "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/errors"
)

var csvHeader = []string{"id", "tanggal", "referensi", "nama", "nominal", "kanal", "catatan"}

const (
	// MaxImportRows is the documented MVP import limit (data rows, excluding
	// the header). MaxCSVBytes is the UTF-8 CSV payload limit.
	MaxImportRows = 500
	MaxCSVBytes   = 1 << 20
	MaxEntries    = 10_000
	// Keep exception responses bounded when many transactions share the same
	// reference, date, channel, and amount. The UI can still show the full
	// candidate_count and fall back to manual allocation.
	MaxCandidateIDs = 20
)

func parseCSV(kind, raw string) ([]models.Entry, int64, string, error) {
	if !utf8.ValidString(raw) {
		return nil, 0, "", pkgErr.New(pkgErr.ErrInvalidRequest, "Simpan CSV dengan encoding UTF-8, lalu coba lagi.")
	}
	if kind != "sale" && kind != "receipt" {
		return nil, 0, "", pkgErr.New(pkgErr.ErrInvalidRequest, "Pilih jenis penjualan atau penerimaan.")
	}
	if len(raw) > MaxCSVBytes {
		return nil, 0, "", pkgErr.New(pkgErr.ErrInvalidRequest, "CSV maksimal 1 MB dan 500 baris.")
	}

	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(raw, "\ufeff")))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = 7

	header, err := reader.Read()
	if err != nil {
		return nil, 0, "", pkgErr.New(pkgErr.ErrInvalidRequest, "CSV kosong atau header tidak valid.")
	}
	for i, value := range header {
		if strings.ToLower(strings.TrimSpace(value)) != csvHeader[i] {
			return nil, 0, "", pkgErr.New(pkgErr.ErrInvalidRequest, "Gunakan header: id,tanggal,referensi,nama,nominal,kanal,catatan")
		}
	}

	entries := make([]models.Entry, 0)
	seen := map[string]bool{}
	var total int64
	for line := 2; ; line++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, "", pkgErr.New(pkgErr.ErrInvalidRequest, fmt.Sprintf("Baris %d: periksa jumlah kolom dan tanda petik.", line))
		}
		if len(entries) >= MaxImportRows {
			return nil, 0, "", pkgErr.New(pkgErr.ErrInvalidRequest, "Satu impor maksimal 500 baris.")
		}

		for i := range row {
			row[i] = strings.TrimSpace(row[i])
			if len(row[i]) > 400 || strings.ContainsRune(row[i], 0) {
				return nil, 0, "", pkgErr.New(pkgErr.ErrInvalidRequest, fmt.Sprintf("Baris %d terlalu panjang.", line))
			}
		}
		if row[0] == "" || len(row[0]) > 80 || seen[row[0]] {
			return nil, 0, "", pkgErr.New(pkgErr.ErrInvalidRequest, fmt.Sprintf("Baris %d: ID kosong, terlalu panjang, atau berulang dalam file.", line))
		}
		seen[row[0]] = true
		if _, err := time.Parse("2006-01-02", row[1]); err != nil {
			return nil, 0, "", pkgErr.New(pkgErr.ErrInvalidRequest, fmt.Sprintf("Baris %d: tanggal harus YYYY-MM-DD.", line))
		}
		amount, err := strconv.ParseInt(row[4], 10, 64)
		if err != nil || amount <= 0 || amount > 1_000_000_000_000 {
			return nil, 0, "", pkgErr.New(pkgErr.ErrInvalidRequest, fmt.Sprintf("Baris %d: nominal harus bilangan Rupiah positif tanpa titik/koma, maksimal 1 triliun.", line))
		}
		if row[3] == "" || row[5] == "" {
			return nil, 0, "", pkgErr.New(pkgErr.ErrInvalidRequest, fmt.Sprintf("Baris %d: nama dan kanal wajib diisi.", line))
		}

		entries = append(entries, models.Entry{
			ID:        newID(),
			SourceID:  row[0],
			Kind:      kind,
			Date:      row[1],
			Reference: row[2],
			Name:      row[3],
			Amount:    amount,
			Channel:   strings.ToLower(row[5]),
			Note:      row[6],
		})
		total += amount
	}
	if len(entries) == 0 {
		return nil, 0, "", pkgErr.New(pkgErr.ErrInvalidRequest, "CSV belum memiliki baris transaksi.")
	}

	return entries, total, checksum(kind, raw), nil
}

func checksum(kind, raw string) string {
	sum := sha256.Sum256([]byte(kind + raw))
	return hex.EncodeToString(sum[:])
}

func sameEntry(a, b models.Entry) bool {
	return a.SourceID == b.SourceID && a.Kind == b.Kind && a.Date == b.Date &&
		a.Reference == b.Reference && a.Name == b.Name && a.Amount == b.Amount &&
		a.Channel == b.Channel && a.Note == b.Note && !a.Canceled
}

func filterImportRows(entries []models.Entry, existing map[string]models.Entry) ([]models.Entry, int, error) {
	newRows := make([]models.Entry, 0, len(entries))
	skipped := 0
	for _, entry := range entries {
		old, found := existing[entry.SourceID]
		if !found {
			newRows = append(newRows, entry)
			continue
		}
		if !sameEntry(old, entry) || old.Canceled {
			return nil, 0, pkgErr.New(pkgErr.ErrConflict, "ID sumber "+entry.SourceID+" sudah ada dengan data berbeda atau telah dibatalkan. Impor tidak disimpan.")
		}
		skipped++
	}
	return newRows, skipped, nil
}

func remaining(entry models.Entry, allocations []models.Allocation) int64 {
	if entry.Canceled {
		return 0
	}
	value := entry.Amount
	for _, allocation := range allocations {
		if allocation.Reversed {
			continue
		}
		switch entry.Kind {
		case "sale":
			if allocation.SaleID == entry.ID {
				value -= allocation.Amount + allocation.Fee
			}
		case "receipt":
			if allocation.ReceiptID == entry.ID {
				value -= allocation.Amount
			}
		}
	}
	return value
}

func suggestions(entries []models.Entry, allocations []models.Allocation) []models.Match {
	out := make([]models.Match, 0)
	type key struct {
		Reference string
		Channel   string
		Date      string
		Amount    int64
	}
	keyFor := func(entry models.Entry, date string) key {
		return key{
			Reference: strings.ToLower(strings.TrimSpace(entry.Reference)),
			Channel:   entry.Channel,
			Date:      date,
			Amount:    entry.Amount,
		}
	}

	allocated := map[string]int64{}
	for _, allocation := range allocations {
		if allocation.Reversed {
			continue
		}
		allocated[allocation.SaleID] += allocation.Amount + allocation.Fee
		allocated[allocation.ReceiptID] += allocation.Amount
	}

	sales := map[key][]models.Entry{}
	receipts := map[key][]models.Entry{}
	for _, entry := range entries {
		if entry.Canceled || allocated[entry.ID] != 0 || strings.TrimSpace(entry.Reference) == "" || isCashSale(entry) || isNonOperationalReceipt(entry) {
			continue
		}
		candidate := keyFor(entry, entry.Date)
		switch entry.Kind {
		case "sale":
			sales[candidate] = append(sales[candidate], entry)
		case "receipt":
			receipts[candidate] = append(receipts[candidate], entry)
		}
	}

	for _, sale := range entries {
		if sale.Kind != "sale" || sale.Canceled || allocated[sale.ID] != 0 || strings.TrimSpace(sale.Reference) == "" || isCashSale(sale) {
			continue
		}
		saleDate, err := time.Parse("2006-01-02", sale.Date)
		if err != nil {
			continue
		}

		count := 0
		var receipt models.Entry
		for day := 0; day <= 3; day++ {
			bucket := receipts[keyFor(sale, saleDate.AddDate(0, 0, day).Format("2006-01-02"))]
			count += len(bucket)
			if len(bucket) == 1 {
				receipt = bucket[0]
			}
		}
		if count != 1 {
			continue
		}

		receiptDate, _ := time.Parse("2006-01-02", receipt.Date)
		reverseCount := 0
		for day := 0; day <= 3; day++ {
			reverseCount += len(sales[keyFor(receipt, receiptDate.AddDate(0, 0, -day).Format("2006-01-02"))])
		}
		if reverseCount == 1 {
			out = append(out, models.Match{
				Sale:    sale,
				Receipt: receipt,
				Reason:  "Referensi, kanal, dan nominal sama; hanya satu kandidat di kedua sisi (maksimal 3 hari).",
			})
		}
	}

	return out
}

// exceptions returns the items that still need a human decision. It uses the
// same matching rules as suggestions, but keeps ambiguous candidates visible
// instead of dropping them.
func exceptions(entries []models.Entry, allocations []models.Allocation) []models.Exception {
	remainingByID := make(map[string]int64, len(entries))
	activeByID := make(map[string]int, len(entries))
	for _, entry := range entries {
		remainingByID[entry.ID] = entry.Amount
	}
	for _, allocation := range allocations {
		if allocation.Reversed {
			continue
		}
		if _, ok := remainingByID[allocation.SaleID]; ok {
			remainingByID[allocation.SaleID] -= allocation.Amount + allocation.Fee
			activeByID[allocation.SaleID]++
		}
		if _, ok := remainingByID[allocation.ReceiptID]; ok {
			remainingByID[allocation.ReceiptID] -= allocation.Amount
			activeByID[allocation.ReceiptID]++
		}
	}

	type candidateKey struct {
		Reference string
		Channel   string
		Date      string
		Amount    int64
	}
	keyFor := func(entry models.Entry, date string) candidateKey {
		return candidateKey{
			Reference: strings.ToLower(strings.TrimSpace(entry.Reference)),
			Channel:   entry.Channel,
			Date:      date,
			Amount:    entry.Amount,
		}
	}
	sales := make(map[candidateKey][]models.Entry)
	receipts := make(map[candidateKey][]models.Entry)
	for _, entry := range entries {
		if entry.Canceled || strings.TrimSpace(entry.Reference) == "" || remainingByID[entry.ID] <= 0 || isCashSale(entry) || isNonOperationalReceipt(entry) {
			continue
		}
		switch entry.Kind {
		case "sale":
			sales[keyFor(entry, entry.Date)] = append(sales[keyFor(entry, entry.Date)], entry)
		case "receipt":
			receipts[keyFor(entry, entry.Date)] = append(receipts[keyFor(entry, entry.Date)], entry)
		}
	}
	// A one-sided bucket is not enough to confirm a pair: both directions
	// must be unique. Reuse the matcher so exceptions cannot advertise a
	// suggestion that the suggestions endpoint refuses to emit.
	suggested := make(map[string]bool)
	for _, match := range suggestions(entries, allocations) {
		suggested[match.Sale.ID] = true
		suggested[match.Receipt.ID] = true
	}

	out := make([]models.Exception, 0)
	for _, entry := range entries {
		if entry.Canceled || isCashSale(entry) || isNonOperationalReceipt(entry) {
			continue
		}
		remainingAmount := remainingByID[entry.ID]
		if remainingAmount <= 0 {
			continue
		}

		status := "unmatched"
		reason := ""
		candidateIDs := make([]string, 0)
		candidateCount := 0
		candidateIDsTruncated := false
		suggestionAvailable := false
		if strings.TrimSpace(entry.Reference) != "" {
			date, err := time.Parse("2006-01-02", entry.Date)
			if err == nil {
				seenCandidates := make(map[string]struct{})
				addCandidate := func(candidate models.Entry) {
					if _, seen := seenCandidates[candidate.ID]; seen {
						return
					}
					seenCandidates[candidate.ID] = struct{}{}
					candidateCount++
					if len(candidateIDs) < MaxCandidateIDs {
						candidateIDs = append(candidateIDs, candidate.ID)
					} else {
						candidateIDsTruncated = true
					}
				}
				for day := 0; day <= 3; day++ {
					var dateKey string
					if entry.Kind == "sale" {
						dateKey = date.AddDate(0, 0, day).Format("2006-01-02")
					} else {
						dateKey = date.AddDate(0, 0, -day).Format("2006-01-02")
					}
					key := keyFor(entry, dateKey)
					if entry.Kind == "sale" {
						for _, candidate := range receipts[key] {
							addCandidate(candidate)
						}
					} else {
						for _, candidate := range sales[key] {
							addCandidate(candidate)
						}
					}
				}
			}
		}
		switch {
		case activeByID[entry.ID] > 0:
			status = "partial"
			reason = partialReason(entry, remainingAmount)
		case strings.TrimSpace(entry.Reference) == "":
			reason = "Referensi kosong; periksa dan alokasikan manual."
		default:
			switch {
			case candidateCount == 0:
				reason = "Tidak ada kandidat otomatis; periksa manual."
			case candidateCount == 1 && suggested[entry.ID]:
				suggestionAvailable = true
				reason = "Kandidat unik tersedia; belum dikonfirmasi."
			case candidateCount == 1:
				status = "ambiguous"
				reason = "Kandidat belum unik di kedua sisi; periksa manual."
			default:
				status = "ambiguous"
				reason = fmt.Sprintf("%d kandidat; pasangan belum unik.", candidateCount)
			}
		}

		out = append(out, models.Exception{
			Entry:                 entry,
			Status:                status,
			Remaining:             remainingAmount,
			CandidateIDs:          candidateIDs,
			CandidateCount:        candidateCount,
			CandidateIDsTruncated: candidateIDsTruncated,
			SuggestionAvailable:   suggestionAvailable,
			Reason:                reason,
			FeeHint:               feeHintFor(entry, entries),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Entry.Date == out[j].Entry.Date {
			return out[i].Entry.ID < out[j].Entry.ID
		}
		return out[i].Entry.Date < out[j].Entry.Date
	})
	return out
}

func partialReason(entry models.Entry, remainingAmount int64) string {
	if entry.Kind == "sale" {
		return fmt.Sprintf("Pembayaran sebagian; sisa tagihan Rp%d.", remainingAmount)
	}
	return fmt.Sprintf("Penerimaan sebagian; Rp%d belum teralokasi.", remainingAmount)
}

func validateAllocation(sale, receipt models.Entry, active []models.Allocation, amount, fee int64, reason string) error {
	if amount <= 0 || fee < 0 || amount > 1_000_000_000_000 || fee > 1_000_000_000_000 {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "Nominal harus positif dan biaya tidak boleh negatif.")
	}
	if sale.Kind != "sale" || receipt.Kind != "receipt" || sale.Canceled || receipt.Canceled {
		return pkgErr.New(pkgErr.ErrNotFound, "Penjualan atau penerimaan tidak ditemukan.")
	}
	if isCashSale(sale) {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "Penjualan tunai sudah lunas dan tidak perlu dipasangkan.")
	}
	if isNonOperationalReceipt(receipt) {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "Penerimaan ini non-operasional, bukan pembayaran pelanggan.")
	}
	remainingSale := remaining(sale, active)
	remainingReceipt := remaining(receipt, active)
	if amount+fee > remainingSale || amount > remainingReceipt {
		return pkgErr.New(pkgErr.ErrConflict, "Alokasi melebihi sisa tagihan atau penerimaan. Muat ulang dan periksa sisa terbaru.")
	}
	if fee > 0 && utf8.RuneCountInString(strings.TrimSpace(reason)) < 5 {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "Jelaskan biaya layanan minimal 5 karakter.")
	}
	if len([]rune(reason)) > 400 || strings.ContainsRune(reason, 0) {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "Catatan alokasi maksimal 400 karakter dan tidak boleh memuat karakter kosong.")
	}
	return nil
}

func buildExportCSV(entries []models.Entry, allocations []models.Allocation) ([]byte, error) {
	ordered := append([]models.Entry(nil), entries...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Date == ordered[j].Date {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].Date < ordered[j].Date
	})

	var buffer bytes.Buffer
	buffer.WriteString("\ufeff")
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"id", "jenis", "tanggal", "referensi", "nama", "nominal", "sisa", "kanal", "status", "catatan"}); err != nil {
		return nil, err
	}
	for _, entry := range ordered {
		status := "aktif"
		if entry.Canceled {
			status = "batal"
		}
		row := []string{
			safeCell(entry.SourceID),
			safeCell(entry.Kind),
			safeCell(entry.Date),
			safeCell(entry.Reference),
			safeCell(entry.Name),
			strconv.FormatInt(entry.Amount, 10),
			strconv.FormatInt(remaining(entry, allocations), 10),
			safeCell(entry.Channel),
			status,
			safeCell(entry.Note),
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func safeCell(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}
