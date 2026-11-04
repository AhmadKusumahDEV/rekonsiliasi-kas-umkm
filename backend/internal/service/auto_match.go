package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/ai"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	sqlc "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/repository/rekonsiliasi_repository_sqlc"
	"github.com/jackc/pgx/v5"
)

// auto-match weights. The sum is 1.0 so a perfect pair scores 1.0.
const (
	weightReference = 0.35
	weightAmount    = 0.25
	weightChannel   = 0.15
	weightDate      = 0.15
	weightName      = 0.10
)

// reviewFloor is the score below which a candidate is not even considered.
// Candidates between reviewFloor and the auto threshold still block automatic
// pairing because they represent a plausible alternative.
const reviewFloor = 0.60

func autoMatchEnabled() bool {
	value := strings.TrimSpace(os.Getenv("MATCH_AUTO_APPLY"))
	if value == "" {
		return true
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return true
	}
	return enabled
}

func autoMatchThreshold() float64 {
	value := strings.TrimSpace(os.Getenv("MATCH_AUTO_THRESHOLD"))
	if value == "" {
		return 0.95
	}
	threshold, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(threshold) || math.IsInf(threshold, 0) || threshold <= 0 || threshold > 1 {
		return 0.95
	}
	return threshold
}

// normalizeReference keeps letters and digits so "TRF/INV-102" matches "inv102".
func normalizeReference(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func normalizeName(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func nameSimilarity(a, b string) float64 {
	x, y := normalizeName(a), normalizeName(b)
	if x == "" || y == "" {
		return 0
	}
	if x == y {
		return 1
	}
	if strings.Contains(x, y) || strings.Contains(y, x) {
		return 0.7
	}
	xf := strings.Fields(x)
	yf := strings.Fields(y)
	if len(xf) > 0 && len(yf) > 0 && xf[0] == yf[0] {
		return 0.4
	}
	return 0
}

func dateDistanceDays(a, b string) (int, bool) {
	x, err := time.Parse("2006-01-02", a)
	if err != nil {
		return 0, false
	}
	y, err := time.Parse("2006-01-02", b)
	if err != nil {
		return 0, false
	}
	days := int(math.Abs(x.Sub(y).Hours() / 24))
	return days, true
}

// scorePair returns a confidence score between 0 and 1 plus the reasons that
// contributed to it. Amounts must match for an automatic decision; the caller
// enforces that rule separately.
func scorePair(sale, receipt models.Entry) (float64, []string) {
	score := 0.0
	reasons := make([]string, 0, 4)

	saleRef, receiptRef := normalizeReference(sale.Reference), normalizeReference(receipt.Reference)
	if saleRef != "" && saleRef == receiptRef {
		score += weightReference
		reasons = append(reasons, "referensi sama")
	} else if saleRef != "" && receiptRef != "" && (strings.Contains(receiptRef, saleRef) || strings.Contains(saleRef, receiptRef)) {
		score += weightReference * 0.6
		reasons = append(reasons, "referensi mirip")
	}

	if sale.Amount == receipt.Amount && sale.Amount > 0 {
		score += weightAmount
		reasons = append(reasons, "nominal sama")
	}

	if strings.EqualFold(strings.TrimSpace(sale.Channel), strings.TrimSpace(receipt.Channel)) {
		score += weightChannel
		reasons = append(reasons, "kanal sama")
	}

	if days, ok := dateDistanceDays(sale.Date, receipt.Date); ok {
		switch {
		case days == 0:
			score += weightDate
			reasons = append(reasons, "tanggal sama")
		case days <= 3:
			score += weightDate * 0.8
			reasons = append(reasons, fmt.Sprintf("selisih %d hari", days))
		case days <= 7:
			score += weightDate * 0.4
			reasons = append(reasons, fmt.Sprintf("selisih %d hari", days))
		}
	}

	if similarity := nameSimilarity(sale.Name, receipt.Name); similarity > 0 {
		score += weightName * similarity
		reasons = append(reasons, "nama mirip")
	}

	if score > 1 {
		score = 1
	}
	return score, reasons
}

type autoPair struct {
	sale    models.Entry
	receipt models.Entry
	score   float64
	reasons []string
}

// autoMatchPairs picks pairs that clear the threshold and are unambiguous on
// both sides. Any tie or missing reference stays in the review queue.
func autoMatchPairs(entries []models.Entry, threshold float64) []autoPair {
	sales := make([]models.Entry, 0, len(entries))
	receipts := make([]models.Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.Canceled || strings.TrimSpace(entry.Reference) == "" || entry.Amount <= 0 || isCashSale(entry) || isNonOperationalReceipt(entry) {
			continue
		}
		switch entry.Kind {
		case "sale":
			sales = append(sales, entry)
		case "receipt":
			receipts = append(receipts, entry)
		}
	}
	if len(sales) == 0 || len(receipts) == 0 || len(sales)*len(receipts) > 40000 {
		return nil
	}

	type candidate struct {
		receiptIndex int
		score        float64
		reasons      []string
	}
	bestBySale := make([][]candidate, len(sales))
	bestByReceipt := make([][]int, len(receipts))
	for i, sale := range sales {
		for j, receipt := range receipts {
			if sale.Amount != receipt.Amount {
				continue
			}
			score, reasons := scorePair(sale, receipt)
			if score < reviewFloor {
				continue
			}
			bestBySale[i] = append(bestBySale[i], candidate{receiptIndex: j, score: score, reasons: reasons})
			bestByReceipt[j] = append(bestByReceipt[j], i)
		}
	}

	pairs := make([]autoPair, 0)
	for i, candidates := range bestBySale {
		if len(candidates) != 1 {
			continue
		}
		chosen := candidates[0]
		if chosen.score < threshold {
			continue
		}
		if len(bestByReceipt[chosen.receiptIndex]) != 1 {
			continue
		}
		pairs = append(pairs, autoPair{
			sale:    sales[i],
			receipt: receipts[chosen.receiptIndex],
			score:   chosen.score,
			reasons: chosen.reasons,
		})
	}
	return pairs
}

type autoMatchResult struct {
	Rule int
	AI   int
}

func aiMinScore() float64 {
	value := strings.TrimSpace(os.Getenv("AI_MIN_SCORE"))
	if value == "" {
		return 0.90
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed < reviewFloor || parsed > 1 {
		return 0.90
	}
	return parsed
}

func (s *Service) autoMatchWorkspace(ctx context.Context, user models.User, importID string) (autoMatchResult, error) {
	result := autoMatchResult{}
	if !autoMatchEnabled() || s.db == nil {
		return result, nil
	}
	threshold := autoMatchThreshold()

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return result, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)
	if err = lockWorkspace(ctx, queries, user.WorkspaceID); err != nil {
		return result, err
	}
	rows, err := queries.ListUmkmUnallocatedEntries(ctx, user.WorkspaceID)
	if err != nil {
		return result, fmt.Errorf("list unallocated entries: %w", err)
	}
	entries := make([]models.Entry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, mapEntry(row))
	}
	pairs := autoMatchPairs(entries, threshold)
	if importID != "" {
		relevant := make([]autoPair, 0, len(pairs))
		for _, pair := range pairs {
			if pair.sale.BatchID == importID || pair.receipt.BatchID == importID {
				relevant = append(relevant, pair)
			}
		}
		pairs = relevant
	}

	used := make(map[string]bool)
	for _, pair := range pairs {
		if _, err = queries.CreateUmkmAllocation(ctx, sqlc.CreateUmkmAllocationParams{
			ID:             newID(),
			WorkspaceID:    user.WorkspaceID,
			SaleEntryID:    pair.sale.ID,
			ReceiptEntryID: pair.receipt.ID,
			Amount:         pair.sale.Amount,
			Fee:            0,
			Reason:         fmt.Sprintf("Cocok otomatis (skor %.2f): %s", pair.score, strings.Join(pair.reasons, ", ")),
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return autoMatchResult{}, fmt.Errorf("create auto allocation: %w", err)
		}
		used[pair.sale.ID] = true
		used[pair.receipt.ID] = true
		result.Rule++
	}

	if s.aiReviewer != nil {
		aiCount, aiErr := s.applyAIPairs(ctx, queries, user, entries, used, threshold)
		if aiErr == nil {
			result.AI = aiCount
		}
	}
	if result.Rule == 0 && result.AI == 0 {
		return result, tx.Commit(ctx)
	}
	version, err := insertAuditAndBump(ctx, queries, user, "Simpan otomatis", fmt.Sprintf("%d pasangan aturan, %d disarankan AI", result.Rule, result.AI), "import", importID)
	if err != nil {
		return autoMatchResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return autoMatchResult{}, err
	}
	s.publishState(user.WorkspaceID, version)
	return result, nil
}

func (s *Service) applyAIPairs(ctx context.Context, queries *sqlc.Queries, user models.User, entries []models.Entry, used map[string]bool, threshold float64) (int, error) {
	type scored struct {
		sale    models.Entry
		receipt models.Entry
		score   float64
		reasons []string
	}
	candidates := make([]scored, 0)
	for _, sale := range entries {
		if sale.Kind != "sale" || used[sale.ID] || isCashSale(sale) {
			continue
		}
		for _, receipt := range entries {
			if receipt.Kind != "receipt" || used[receipt.ID] || isNonOperationalReceipt(receipt) {
				continue
			}
			if sale.Amount != receipt.Amount || sale.Amount <= 0 {
				continue
			}
			score, reasons := scorePair(sale, receipt)
			if score < reviewFloor || score >= threshold {
				continue
			}
			candidates = append(candidates, scored{sale: sale, receipt: receipt, score: score, reasons: reasons})
		}
	}
	if len(candidates) == 0 {
		return 0, nil
	}
	saleCounts := make(map[string]int, len(candidates))
	receiptCounts := make(map[string]int, len(candidates))
	for _, candidate := range candidates {
		saleCounts[candidate.sale.ID]++
		receiptCounts[candidate.receipt.ID]++
	}
	unique := make([]scored, 0, len(candidates))
	for _, candidate := range candidates {
		if saleCounts[candidate.sale.ID] == 1 && receiptCounts[candidate.receipt.ID] == 1 {
			unique = append(unique, candidate)
		}
	}
	if len(unique) == 0 {
		return 0, nil
	}
	candidates = unique
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	if len(candidates) > 20 {
		candidates = candidates[:20]
	}
	payload := make([]ai.Candidate, 0, len(candidates))
	index := make(map[string]scored, len(candidates))
	for _, candidate := range candidates {
		key := candidate.sale.ID + "|" + candidate.receipt.ID
		index[key] = candidate
		payload = append(payload, ai.Candidate{
			SaleID:        candidate.sale.ID,
			ReceiptID:     candidate.receipt.ID,
			Score:         candidate.score,
			SaleRef:       candidate.sale.Reference,
			ReceiptRef:    candidate.receipt.Reference,
			SaleDate:      candidate.sale.Date,
			ReceiptDate:   candidate.receipt.Date,
			SaleAmount:    candidate.sale.Amount,
			ReceiptAmount: candidate.receipt.Amount,
		})
	}
	verdicts, err := s.aiReviewer.Review(ctx, payload)
	if err != nil {
		return 0, err
	}
	minimum := aiMinScore()
	created := 0
	for _, verdict := range verdicts {
		if !verdict.Match {
			continue
		}
		key := verdict.SaleID + "|" + verdict.ReceiptID
		candidate, ok := index[key]
		if !ok || candidate.score < minimum || used[candidate.sale.ID] || used[candidate.receipt.ID] {
			continue
		}
		reason := strings.TrimSpace(verdict.Reason)
		if reason == "" {
			reason = "kandidat paling sesuai"
		}
		if _, err = queries.CreateUmkmAllocation(ctx, sqlc.CreateUmkmAllocationParams{
			ID:             newID(),
			WorkspaceID:    user.WorkspaceID,
			SaleEntryID:    candidate.sale.ID,
			ReceiptEntryID: candidate.receipt.ID,
			Amount:         candidate.sale.Amount,
			Fee:            0,
			Reason:         fmt.Sprintf("Disarankan AI (skor aturan %.2f): %s", candidate.score, reason),
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return created, err
		}
		used[candidate.sale.ID] = true
		used[candidate.receipt.ID] = true
		created++
	}
	return created, nil
}
