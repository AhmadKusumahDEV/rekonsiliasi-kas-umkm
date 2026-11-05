package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/ai"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	sqlc "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/repository/rekonsiliasi_repository_sqlc"
	"github.com/jackc/pgx/v5"
)

type aiScoredPair struct {
	sale    models.Entry
	receipt models.Entry
	score   float64
}

func shortImportRef(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func aiErrorKind(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "status 401"), strings.Contains(msg, "status 403"):
		return "auth"
	case strings.Contains(msg, "status 429"):
		return "rate_limited"
	case strings.Contains(msg, "status 5"):
		return "upstream"
	case strings.Contains(msg, "json"):
		return "invalid_response"
	default:
		return "request_failed"
	}
}

func sanitizeAIReason(value string) string {
	if !utf8.ValidString(value) {
		return ""
	}
	value = strings.Map(func(r rune) rune {
		if r == 0 || unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return ' '
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > 120 {
		runes = runes[:120]
	}
	return string(runes)
}

func safeReference(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return -1
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > 80 {
		runes = runes[:80]
	}
	return string(runes)
}

func (s *Service) applyAIReview(ctx context.Context, tx pgx.Tx, queries *sqlc.Queries, user models.User, entries []models.Entry, used map[string]bool, threshold float64, importID string) (int, string, error) {
	sales, receipts := 0, 0
	for _, entry := range entries {
		switch entry.Kind {
		case "sale":
			sales++
		case "receipt":
			receipts++
		}
	}
	if sales*receipts > 40000 {
		s.log().WarnContext(ctx, "ai review skipped", "event", "ai_review_skipped", "ai_reason", "too_many_candidates")
		return 0, "skipped", nil
	}

	all := make([]aiScoredPair, 0)
	for _, sale := range entries {
		if sale.Kind != "sale" || used[sale.ID] || isCashSale(sale) {
			continue
		}
		for _, receipt := range entries {
			if receipt.Kind != "receipt" || used[receipt.ID] || isNonOperationalReceipt(receipt) {
				continue
			}
			if importID != "" && sale.BatchID != importID && receipt.BatchID != importID {
				continue
			}
			if sale.Amount != receipt.Amount || sale.Amount <= 0 {
				continue
			}
			score, _ := scorePair(sale, receipt)
			if score < reviewFloor {
				continue
			}
			all = append(all, aiScoredPair{sale: sale, receipt: receipt, score: score})
		}
	}
	if len(all) == 0 {
		return 0, "no_candidates", nil
	}

	saleCounts := make(map[string]int, len(all))
	receiptCounts := make(map[string]int, len(all))
	for _, pair := range all {
		saleCounts[pair.sale.ID]++
		receiptCounts[pair.receipt.ID]++
	}
	minimum := aiMinScore()
	if minimum >= threshold {
		s.log().WarnContext(ctx, "ai review skipped", "event", "ai_review_skipped", "ai_reason", "min_score_not_below_threshold")
		return 0, "skipped", nil
	}
	candidates := make([]aiScoredPair, 0, len(all))
	for _, pair := range all {
		if pair.score < minimum || pair.score >= threshold {
			continue
		}
		if saleCounts[pair.sale.ID] != 1 || receiptCounts[pair.receipt.ID] != 1 {
			continue
		}
		candidates = append(candidates, pair)
	}
	if len(candidates) == 0 {
		return 0, "no_candidates", nil
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	if len(candidates) > 20 {
		candidates = candidates[:20]
	}

	payload := make([]ai.Candidate, 0, len(candidates))
	index := make(map[string]aiScoredPair, len(candidates))
	for _, pair := range candidates {
		key := pair.sale.ID + "|" + pair.receipt.ID
		index[key] = pair
		nameSim := nameSimilarity(pair.sale.Name, pair.receipt.Name)
		dateDelta := 0
		if days, ok := dateDistanceDays(pair.sale.Date, pair.receipt.Date); ok {
			dateDelta = days
		}
		payload = append(payload, ai.Candidate{
			SaleID:         pair.sale.ID,
			ReceiptID:      pair.receipt.ID,
			Score:          pair.score,
			SaleRef:        safeReference(pair.sale.Reference),
			ReceiptRef:     safeReference(pair.receipt.Reference),
			SaleDate:       pair.sale.Date,
			ReceiptDate:    pair.receipt.Date,
			SaleAmount:     pair.sale.Amount,
			ReceiptAmount:  pair.receipt.Amount,
			Channel:        pair.sale.Channel,
			NameSimilarity: nameSim,
			DateDeltaDays:  dateDelta,
			ReferenceEqual: normalizeReference(pair.sale.Reference) == normalizeReference(pair.receipt.Reference),
		})
	}
	started := time.Now()
	s.log().InfoContext(ctx, "ai review start", "event", "ai_review_start", "ai_candidates", len(payload), "import_ref", shortImportRef(importID))
	verdicts, err := s.aiReviewer.Review(ctx, payload)
	if err != nil {
		return 0, "failed", err
	}
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
		reason := sanitizeAIReason(verdict.Reason)
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
			return 0, "failed", err
		}
		used[candidate.sale.ID] = true
		used[candidate.receipt.ID] = true
		created++
	}
	s.log().InfoContext(ctx, "ai review completed", "event", "ai_review_result", "ai_verdicts", len(verdicts), "ai_applied", created, "duration_ms", time.Since(started).Milliseconds(), "import_ref", shortImportRef(importID))
	return created, "ok", nil
}
