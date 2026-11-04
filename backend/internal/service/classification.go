package service

import (
	"fmt"
	"os"
	"strings"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
)

func feeSource(channel string) (string, string) {
	value := normalizePhrase(channel)
	switch {
	case strings.Contains(value, "MARKETPLACE"), strings.Contains(value, "SHOPEE"), strings.Contains(value, "TOKOPEDIA"), strings.Contains(value, "TIKTOK"), strings.Contains(value, "LAZADA"):
		return "marketplace", "Biaya marketplace"
	case strings.Contains(value, "QRIS"), strings.Contains(value, "TRANSFER"), strings.Contains(value, "BANK"), strings.Contains(value, "WALLET"), strings.Contains(value, "GOPAY"), strings.Contains(value, "OVO"), strings.Contains(value, "DANA"), strings.Contains(value, "KARTU"):
		return "bank", "Biaya bank atau penyedia pembayaran"
	default:
		return "unknown", "Selisih nominal"
	}
}

// feeHintFor suggests a fee when exactly one receipt with the same reference is
// slightly smaller than the sale. Anything larger or ambiguous is left alone.
func feeHintFor(entry models.Entry, entries []models.Entry) *models.FeeHint {
	if entry.Kind != "sale" || strings.TrimSpace(entry.Reference) == "" || entry.Amount <= 0 {
		return nil
	}
	reference := normalizeReference(entry.Reference)
	var matched *models.Entry
	count := 0
	for i := range entries {
		candidate := entries[i]
		if candidate.Kind != "receipt" || candidate.Canceled || isNonOperationalReceipt(candidate) {
			continue
		}
		if normalizeReference(candidate.Reference) != reference || candidate.Amount >= entry.Amount {
			continue
		}
		delta := entry.Amount - candidate.Amount
		if delta > max(int64(25000), entry.Amount/20) {
			continue
		}
		count++
		if count == 1 {
			copy := candidate
			matched = &copy
		}
	}
	if count != 1 || matched == nil {
		return nil
	}
	delta := entry.Amount - matched.Amount
	source, label := feeSource(matched.Channel)
	confidence := "sedang"
	if delta <= max(int64(10000), entry.Amount/100) {
		confidence = "tinggi"
	}
	reason := fmt.Sprintf("Selisih Rp%d dari Rp%d; kanal %s. Periksa sebagai biaya, bukan uang hilang.", delta, entry.Amount, matched.Channel)
	return &models.FeeHint{Amount: delta, Source: source, Label: label, Confidence: confidence, Reason: reason}
}

var defaultNonOperationalKeywords = []string{
	"BUNGA TABUNGAN",
	"BUNGA GIRO",
	"JASA GIRO",
	"BIAYA ADMIN",
	"BIAYA ADM",
	"ADMIN BANK",
	"PAJAK BANK",
	"SETOR PAJAK",
	"PPH",
	"PPN",
	"TRANSFER ANTAR REKENING",
	"TRF ANTAR REKENING",
	"ANTAR REKENING",
	"SESAMA REKENING",
	"SETOR TUNAI",
	"PENARIKAN TUNAI",
	"TARIK TUNAI",
	"CASHBACK",
}

func nonOperationalKeywords() []string {
	keywords := append([]string{}, defaultNonOperationalKeywords...)
	value := strings.TrimSpace(os.Getenv("NON_OPERATIONAL_KEYWORDS_EXTRA"))
	for _, part := range strings.Split(value, ",") {
		if keyword := normalizePhrase(part); keyword != "" {
			keywords = append(keywords, keyword)
		}
	}
	return keywords
}

// isCashSale treats a cash sale as already settled: the money is in hand, so
// it must not wait for a bank receipt.
func isCashSale(entry models.Entry) bool {
	if entry.Kind != "sale" {
		return false
	}
	for _, token := range strings.Fields(normalizePhrase(entry.Channel)) {
		switch strings.Trim(token, "()[].,-") {
		case "TUNAI", "CASH", "KAS", "KASIR":
			return true
		}
	}
	return false
}

// isNonOperationalReceipt marks bank rows such as interest, admin fees, or
// transfers between the owner own accounts. They are real money movements but
// never payments from a customer.
func isNonOperationalReceipt(entry models.Entry) bool {
	if entry.Kind != "receipt" {
		return false
	}
	text := normalizePhrase(entry.Name + " " + entry.Note)
	for _, keyword := range nonOperationalKeywords() {
		if strings.Contains(" "+text+" ", " "+keyword+" ") {
			return true
		}
	}
	return false
}

// normalizePhrase uppercases, removes non-breaking spaces, collapses whitespace,
// and turns hyphens into spaces so phrase matches are predictable.
func normalizePhrase(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\u00a0", " ")
	value = strings.ReplaceAll(value, "\u2013", " ")
	value = strings.ReplaceAll(value, "\u2014", " ")
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.ToUpper(value)
	return strings.Join(strings.Fields(value), " ")
}
