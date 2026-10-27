package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/repository/rekonsiliasi_repository_sqlc"
	pkgErr "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) GetState(ctx context.Context, user models.User) (models.StateResponse, error) {
	if s.db == nil {
		return models.StateResponse{}, pkgErr.New(pkgErr.ErrServiceUnavailable, "Database belum tersedia.")
	}

	queries := sqlc.New(s.db.Pool())
	workspace, err := queries.GetWorkspaceForUser(ctx, sqlc.GetWorkspaceForUserParams{
		WorkspaceID: user.WorkspaceID,
		App:         config.AppKindUMKM,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return models.StateResponse{}, pkgErr.New(pkgErr.ErrNotFound, "Ruang kerja tidak ditemukan.")
	}
	if err != nil {
		return models.StateResponse{}, fmt.Errorf("get workspace: %w", err)
	}

	userRows, err := queries.ListUsersByWorkspace(ctx, sqlc.ListUsersByWorkspaceParams{
		WorkspaceID: user.WorkspaceID,
		App:         config.AppKindUMKM,
	})
	if err != nil {
		return models.StateResponse{}, fmt.Errorf("list users: %w", err)
	}
	eventRows, err := queries.ListAuditEventsByWorkspace(ctx, sqlc.ListAuditEventsByWorkspaceParams{
		WorkspaceID: user.WorkspaceID,
		RowLimit:    120,
	})
	if err != nil {
		return models.StateResponse{}, fmt.Errorf("list audit: %w", err)
	}
	entryRows, err := queries.ListUmkmEntriesByWorkspace(ctx, user.WorkspaceID)
	if err != nil {
		return models.StateResponse{}, fmt.Errorf("list entries: %w", err)
	}
	importRows, err := queries.ListUmkmImportsByWorkspace(ctx, user.WorkspaceID)
	if err != nil {
		return models.StateResponse{}, fmt.Errorf("list imports: %w", err)
	}
	allocationRows, err := queries.ListUmkmAllocationsByWorkspace(ctx, user.WorkspaceID)
	if err != nil {
		return models.StateResponse{}, fmt.Errorf("list allocations: %w", err)
	}

	users := make([]models.User, 0, len(userRows))
	for _, row := range userRows {
		users = append(users, mapUser(row))
	}
	events := make([]models.AuditEvent, 0, len(eventRows))
	for _, row := range eventRows {
		events = append(events, mapAuditEvent(row))
	}
	entries := make([]models.Entry, 0, len(entryRows))
	for _, row := range entryRows {
		entries = append(entries, mapEntry(row))
	}
	imports := make([]models.Import, 0, len(importRows))
	for _, row := range importRows {
		imports = append(imports, mapImport(row))
	}
	allocations := make([]models.Allocation, 0, len(allocationRows))
	for _, row := range allocationRows {
		allocations = append(allocations, mapAllocation(row))
	}

	code := workspace.Code
	if user.Role != "owner" {
		code = ""
	}

	return models.StateResponse{
		Workspace: models.Workspace{
			ID:      workspace.ID,
			Name:    workspace.Name,
			Code:    code,
			Version: workspace.Version,
		},
		State: models.UMKMState{
			Entries:     entries,
			Imports:     imports,
			Allocations: allocations,
		},
		Users:  users,
		Events: events,
	}, nil
}

func (s *Service) GetSuggestions(ctx context.Context, user models.User) ([]models.Match, error) {
	if s.db == nil {
		return nil, pkgErr.New(pkgErr.ErrServiceUnavailable, "Database belum tersedia.")
	}
	rows, err := sqlc.New(s.db.Pool()).ListUmkmUnallocatedEntries(ctx, user.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("list unallocated entries: %w", err)
	}
	entries := make([]models.Entry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, mapEntry(row))
	}
	return suggestions(entries, nil), nil
}

func (s *Service) PreviewImport(kind, raw string) (models.PreviewResponse, error) {
	entries, total, sum, err := parseCSV(kind, raw)
	if err != nil {
		return models.PreviewResponse{}, err
	}
	return models.PreviewResponse{
		Rows:     entries,
		Count:    len(entries),
		Total:    total,
		Checksum: sum,
	}, nil
}

func (s *Service) CommitImport(ctx context.Context, user models.User, req models.ImportRequest) (models.CommitImportResponse, error) {
	if s.db == nil {
		return models.CommitImportResponse{}, pkgErr.New(pkgErr.ErrServiceUnavailable, "Database belum tersedia.")
	}
	entries, _, sum, err := parseCSV(req.Kind, req.CSV)
	if err != nil {
		return models.CommitImportResponse{}, err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return models.CommitImportResponse{}, pkgErr.ErrFailInitializeTx
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)
	if err = lockWorkspace(ctx, queries, user.WorkspaceID); err != nil {
		return models.CommitImportResponse{}, err
	}

	existingImport, err := queries.GetUmkmImportByChecksum(ctx, sqlc.GetUmkmImportByChecksumParams{
		WorkspaceID: user.WorkspaceID,
		Checksum:    sum,
	})
	if err == nil {
		if existingImport.CanceledAt.Valid {
			return models.CommitImportResponse{}, pkgErr.New(pkgErr.ErrConflict, "Impor ini sudah dibatalkan. Gunakan ID sumber baru jika ingin mencatat ulang.")
		}
		version, bumpErr := insertAuditAndBump(ctx, queries, user, "Impor transaksi", "Data CSV tervalidasi")
		if bumpErr != nil {
			return models.CommitImportResponse{}, bumpErr
		}
		if err = tx.Commit(ctx); err != nil {
			return models.CommitImportResponse{}, pkgErr.ErrFailCommitTx
		}
		s.publishState(user.WorkspaceID, version)
		return models.CommitImportResponse{ID: existingImport.ID, Duplicate: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return models.CommitImportResponse{}, fmt.Errorf("get import by checksum: %w", err)
	}

	count, err := queries.CountUmkmEntries(ctx, user.WorkspaceID)
	if err != nil {
		return models.CommitImportResponse{}, fmt.Errorf("count entries: %w", err)
	}
	if count+int64(len(entries)) > 10_000 {
		return models.CommitImportResponse{}, pkgErr.New(pkgErr.ErrInvalidRequest, "Ruang kerja mencapai batas MVP 10.000 transaksi. Ekspor dan hubungi pengelola.")
	}

	sourceIDs := make([]string, 0, len(entries))
	for _, entry := range entries {
		sourceIDs = append(sourceIDs, entry.SourceID)
	}
	existingRows, err := queries.ListUmkmEntriesByKindAndSourceIDs(ctx, sqlc.ListUmkmEntriesByKindAndSourceIDsParams{
		WorkspaceID: user.WorkspaceID,
		Kind:        req.Kind,
		SourceIds:   sourceIDs,
	})
	if err != nil {
		return models.CommitImportResponse{}, fmt.Errorf("list existing entries: %w", err)
	}
	existing := make(map[string]models.Entry, len(existingRows))
	for _, row := range existingRows {
		mapped := mapEntry(row)
		existing[mapped.SourceID] = mapped
	}

	newRows := make([]models.Entry, 0, len(entries))
	skipped := 0
	for _, entry := range entries {
		old, found := existing[entry.SourceID]
		if !found {
			newRows = append(newRows, entry)
			continue
		}
		if !sameEntry(old, entry) || old.Canceled {
			return models.CommitImportResponse{}, pkgErr.New(pkgErr.ErrConflict, "ID sumber "+entry.SourceID+" sudah ada dengan data berbeda atau telah dibatalkan. Impor tidak disimpan.")
		}
		skipped++
	}

	importID := newID()
	if _, err = queries.CreateUmkmImport(ctx, sqlc.CreateUmkmImportParams{
		ID:          importID,
		WorkspaceID: user.WorkspaceID,
		Kind:        req.Kind,
		Checksum:    sum,
		RowCount:    int32(len(newRows)),
	}); err != nil {
		return models.CommitImportResponse{}, fmt.Errorf("create import: %w", err)
	}
	for _, entry := range newRows {
		date, _ := time.Parse("2006-01-02", entry.Date)
		if _, err = queries.CreateUmkmEntry(ctx, sqlc.CreateUmkmEntryParams{
			ID:          entry.ID,
			WorkspaceID: user.WorkspaceID,
			BatchID:     importID,
			SourceID:    entry.SourceID,
			Kind:        entry.Kind,
			OccurredOn:  pgtype.Date{Time: date, Valid: true},
			Reference:   entry.Reference,
			Name:        entry.Name,
			Amount:      entry.Amount,
			Channel:     entry.Channel,
			Note:        entry.Note,
		}); err != nil {
			return models.CommitImportResponse{}, fmt.Errorf("create entry %s: %w", entry.SourceID, err)
		}
	}

	version, bumpErr := insertAuditAndBump(ctx, queries, user, "Impor transaksi", "Data CSV tervalidasi")
	if bumpErr != nil {
		return models.CommitImportResponse{}, bumpErr
	}
	if err = tx.Commit(ctx); err != nil {
		return models.CommitImportResponse{}, pkgErr.ErrFailCommitTx
	}
	s.publishState(user.WorkspaceID, version)
	return models.CommitImportResponse{ID: importID, Imported: len(newRows), Skipped: skipped}, nil
}

func (s *Service) Allocate(ctx context.Context, user models.User, req models.AllocationRequest) (models.Allocation, error) {
	if s.db == nil {
		return models.Allocation{}, pkgErr.New(pkgErr.ErrServiceUnavailable, "Database belum tersedia.")
	}
	if !validHexID(req.Key) || !validHexID(req.SaleID) || !validHexID(req.ReceiptID) {
		return models.Allocation{}, pkgErr.New(pkgErr.ErrInvalidRequest, "ID permintaan, penjualan, atau penerimaan tidak valid.")
	}
	if req.Amount <= 0 || req.Fee < 0 || req.Amount > 1_000_000_000_000 || req.Fee > 1_000_000_000_000 {
		return models.Allocation{}, pkgErr.New(pkgErr.ErrInvalidRequest, "Nominal harus positif dan biaya tidak boleh negatif.")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return models.Allocation{}, pkgErr.ErrFailInitializeTx
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)
	if err = lockWorkspace(ctx, queries, user.WorkspaceID); err != nil {
		return models.Allocation{}, err
	}

	existing, err := queries.GetUmkmAllocationByID(ctx, sqlc.GetUmkmAllocationByIDParams{
		WorkspaceID: user.WorkspaceID,
		ID:          req.Key,
	})
	if err == nil {
		if existing.ReversedAt.Valid || existing.SaleEntryID != req.SaleID || existing.ReceiptEntryID != req.ReceiptID ||
			existing.Amount != req.Amount || existing.Fee != req.Fee || existing.Reason != req.Reason {
			return models.Allocation{}, pkgErr.New(pkgErr.ErrConflict, "ID permintaan sudah digunakan untuk keputusan berbeda. Buka formulir baru.")
		}
		version, bumpErr := insertAuditAndBump(ctx, queries, user, "Konfirmasi pasangan", "Alokasi penerimaan ke penjualan")
		if bumpErr != nil {
			return models.Allocation{}, bumpErr
		}
		if err = tx.Commit(ctx); err != nil {
			return models.Allocation{}, pkgErr.ErrFailCommitTx
		}
		s.publishState(user.WorkspaceID, version)
		return mapAllocation(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return models.Allocation{}, fmt.Errorf("get allocation by id: %w", err)
	}

	entryRows, err := queries.GetUmkmEntriesByIDs(ctx, sqlc.GetUmkmEntriesByIDsParams{
		WorkspaceID: user.WorkspaceID,
		Ids:         []string{req.SaleID, req.ReceiptID},
	})
	if err != nil {
		return models.Allocation{}, fmt.Errorf("get entries: %w", err)
	}
	var sale, receipt models.Entry
	var foundSale, foundReceipt bool
	for _, row := range entryRows {
		entry := mapEntry(row)
		if entry.ID == req.SaleID && entry.Kind == "sale" {
			sale, foundSale = entry, true
		}
		if entry.ID == req.ReceiptID && entry.Kind == "receipt" {
			receipt, foundReceipt = entry, true
		}
	}
	if !foundSale || !foundReceipt || sale.ID == receipt.ID {
		return models.Allocation{}, pkgErr.New(pkgErr.ErrNotFound, "Penjualan atau penerimaan tidak ditemukan.")
	}

	allocationRows, err := queries.ListUmkmActiveAllocationsByEntryIDs(ctx, sqlc.ListUmkmActiveAllocationsByEntryIDsParams{
		WorkspaceID: user.WorkspaceID,
		EntryIds:    []string{req.SaleID, req.ReceiptID},
	})
	if err != nil {
		return models.Allocation{}, fmt.Errorf("list active allocations: %w", err)
	}
	active := make([]models.Allocation, 0, len(allocationRows))
	for _, row := range allocationRows {
		active = append(active, mapAllocation(row))
	}
	if err = validateAllocation(sale, receipt, active, req.Amount, req.Fee, req.Reason); err != nil {
		return models.Allocation{}, err
	}

	created, err := queries.CreateUmkmAllocation(ctx, sqlc.CreateUmkmAllocationParams{
		ID:             req.Key,
		WorkspaceID:    user.WorkspaceID,
		SaleEntryID:    req.SaleID,
		ReceiptEntryID: req.ReceiptID,
		Amount:         req.Amount,
		Fee:            req.Fee,
		Reason:         req.Reason,
	})
	if err != nil {
		return models.Allocation{}, fmt.Errorf("create allocation: %w", err)
	}
	version, bumpErr := insertAuditAndBump(ctx, queries, user, "Konfirmasi pasangan", "Alokasi penerimaan ke penjualan")
	if bumpErr != nil {
		return models.Allocation{}, bumpErr
	}
	if err = tx.Commit(ctx); err != nil {
		return models.Allocation{}, pkgErr.ErrFailCommitTx
	}
	s.publishState(user.WorkspaceID, version)
	return mapAllocation(created), nil
}

func (s *Service) UndoAllocation(ctx context.Context, user models.User, id string) error {
	if s.db == nil {
		return pkgErr.New(pkgErr.ErrServiceUnavailable, "Database belum tersedia.")
	}
	if !validHexID(id) {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "ID alokasi tidak valid.")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return pkgErr.ErrFailInitializeTx
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)
	if err = lockWorkspace(ctx, queries, user.WorkspaceID); err != nil {
		return err
	}
	allocation, err := queries.GetUmkmAllocationByID(ctx, sqlc.GetUmkmAllocationByIDParams{
		WorkspaceID: user.WorkspaceID,
		ID:          id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return pkgErr.New(pkgErr.ErrNotFound, "Alokasi tidak ditemukan.")
	}
	if err != nil {
		return fmt.Errorf("get allocation: %w", err)
	}
	if !allocation.ReversedAt.Valid {
		if _, err = queries.MarkUmkmAllocationReversed(ctx, sqlc.MarkUmkmAllocationReversedParams{
			WorkspaceID: user.WorkspaceID,
			ID:          id,
		}); err != nil {
			return fmt.Errorf("reverse allocation: %w", err)
		}
	}
	version, bumpErr := insertAuditAndBump(ctx, queries, user, "Batalkan pasangan", "Saldo alokasi dikembalikan; riwayat tetap ada")
	if bumpErr != nil {
		return bumpErr
	}
	if err = tx.Commit(ctx); err != nil {
		return pkgErr.ErrFailCommitTx
	}
	s.publishState(user.WorkspaceID, version)
	return nil
}

func (s *Service) CancelImport(ctx context.Context, user models.User, id string) error {
	if s.db == nil {
		return pkgErr.New(pkgErr.ErrServiceUnavailable, "Database belum tersedia.")
	}
	if !validHexID(id) {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "ID impor tidak valid.")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return pkgErr.ErrFailInitializeTx
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)
	if err = lockWorkspace(ctx, queries, user.WorkspaceID); err != nil {
		return err
	}
	if _, err = queries.GetUmkmImportByID(ctx, sqlc.GetUmkmImportByIDParams{
		WorkspaceID: user.WorkspaceID,
		ID:          id,
	}); errors.Is(err, pgx.ErrNoRows) {
		return pkgErr.New(pkgErr.ErrNotFound, "Impor tidak ditemukan.")
	} else if err != nil {
		return fmt.Errorf("get import: %w", err)
	}

	active, err := queries.CountUmkmActiveAllocationsForImport(ctx, sqlc.CountUmkmActiveAllocationsForImportParams{
		WorkspaceID: user.WorkspaceID,
		ImportID:    id,
	})
	if err != nil {
		return fmt.Errorf("count active allocations: %w", err)
	}
	if active > 0 {
		return pkgErr.New(pkgErr.ErrConflict, "Impor memiliki alokasi aktif. Batalkan pasangan terlebih dahulu.")
	}
	if _, err = queries.CancelUmkmImport(ctx, sqlc.CancelUmkmImportParams{WorkspaceID: user.WorkspaceID, ID: id}); err != nil {
		return fmt.Errorf("cancel import: %w", err)
	}
	if _, err = queries.CancelUmkmImportEntries(ctx, sqlc.CancelUmkmImportEntriesParams{WorkspaceID: user.WorkspaceID, ImportID: id}); err != nil {
		return fmt.Errorf("cancel import entries: %w", err)
	}
	version, bumpErr := insertAuditAndBump(ctx, queries, user, "Batalkan impor", "Baris ditandai batal tanpa menghapus audit")
	if bumpErr != nil {
		return bumpErr
	}
	if err = tx.Commit(ctx); err != nil {
		return pkgErr.ErrFailCommitTx
	}
	s.publishState(user.WorkspaceID, version)
	return nil
}

func (s *Service) ExportCSV(ctx context.Context, user models.User) ([]byte, string, error) {
	if s.db == nil {
		return nil, "", pkgErr.New(pkgErr.ErrServiceUnavailable, "Database belum tersedia.")
	}
	if user.Role != "owner" {
		return nil, "", pkgErr.New(pkgErr.ErrForbidden, "Ekspor hanya tersedia untuk pemilik usaha.")
	}
	queries := sqlc.New(s.db.Pool())
	entryRows, err := queries.ListUmkmEntriesByWorkspace(ctx, user.WorkspaceID)
	if err != nil {
		return nil, "", fmt.Errorf("list entries: %w", err)
	}
	allocationRows, err := queries.ListUmkmAllocationsByWorkspace(ctx, user.WorkspaceID)
	if err != nil {
		return nil, "", fmt.Errorf("list allocations: %w", err)
	}
	entries := make([]models.Entry, 0, len(entryRows))
	for _, row := range entryRows {
		entries = append(entries, mapEntry(row))
	}
	allocations := make([]models.Allocation, 0, len(allocationRows))
	for _, row := range allocationRows {
		allocations = append(allocations, mapAllocation(row))
	}
	data, err := buildExportCSV(entries, allocations)
	if err != nil {
		return nil, "", fmt.Errorf("build export: %w", err)
	}
	return data, "umkm-rekap.csv", nil
}

func (s *Service) Config() models.ConfigResponse {
	return models.ConfigResponse{
		Kind:    config.AppKindUMKM,
		Brand:   "Temu Kas",
		Tagline: "Setiap penerimaan, jelas pasangannya.",
		Roles:   []string{"owner"},
	}
}

func lockWorkspace(ctx context.Context, queries *sqlc.Queries, workspaceID string) error {
	_, err := queries.LockWorkspaceForUpdate(ctx, sqlc.LockWorkspaceForUpdateParams{
		WorkspaceID: workspaceID,
		App:         config.AppKindUMKM,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return pkgErr.New(pkgErr.ErrNotFound, "Ruang kerja tidak ditemukan.")
	}
	if err != nil {
		return fmt.Errorf("lock workspace: %w", err)
	}
	return nil
}

func insertAuditAndBump(ctx context.Context, queries *sqlc.Queries, user models.User, action, detail string) (int64, error) {
	if err := queries.InsertAuditEvent(ctx, sqlc.InsertAuditEventParams{
		WorkspaceID: user.WorkspaceID,
		Actor:       user.Name,
		Action:      action,
		Detail:      detail,
	}); err != nil {
		return 0, fmt.Errorf("insert audit: %w", err)
	}
	version, err := queries.BumpWorkspaceVersion(ctx, user.WorkspaceID)
	if err != nil {
		return 0, fmt.Errorf("bump workspace version: %w", err)
	}
	return version, nil
}

func validHexID(value string) bool {
	if len(value) != 32 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
