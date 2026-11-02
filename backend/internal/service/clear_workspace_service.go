package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/repository/rekonsiliasi_repository_sqlc"
	pkgErr "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/errors"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ClearWorkspaceData deletes the UMKM ledger and import history while keeping
// the account, workspace, settings, and audit trail intact.
func (s *Service) ClearWorkspaceData(ctx context.Context, user models.User, req models.ClearWorkspaceDataRequest) error {
	if s.db == nil {
		return pkgErr.New(pkgErr.ErrServiceUnavailable, "Database belum tersedia.")
	}
	if user.Role != "owner" {
		return pkgErr.New(pkgErr.ErrForbidden, "Hanya pemilik usaha yang dapat mengosongkan data.")
	}
	if strings.TrimSpace(req.Confirm) != "HAPUS" {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "Ketik HAPUS untuk mengonfirmasi.")
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

	currentHash, err := queries.GetUserPasswordHash(ctx, user.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return pkgErr.New(pkgErr.ErrUnauthorized, "Akun tidak ditemukan. Masuk kembali.")
	}
	if err != nil {
		return fmt.Errorf("get password hash for clear data: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.Password)) != nil {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "Kata sandi saat ini belum cocok.")
	}

	allocations, err := queries.DeleteUmkmAllocationsByWorkspace(ctx, user.WorkspaceID)
	if err != nil {
		return fmt.Errorf("clear allocations: %w", err)
	}
	entries, err := queries.DeleteUmkmEntriesByWorkspace(ctx, user.WorkspaceID)
	if err != nil {
		return fmt.Errorf("clear entries: %w", err)
	}
	imports, err := queries.DeleteUmkmImportsByWorkspace(ctx, user.WorkspaceID)
	if err != nil {
		return fmt.Errorf("clear imports: %w", err)
	}
	if _, err = queries.ClearWorkspaceState(ctx, sqlc.ClearWorkspaceStateParams{
		WorkspaceID: user.WorkspaceID,
		App:         config.AppKindUMKM,
	}); err != nil {
		return fmt.Errorf("clear legacy workspace state: %w", err)
	}

	detail := fmt.Sprintf("Data transaksi dikosongkan: %d alokasi, %d entri, %d batch impor.", allocations, entries, imports)
	version, err := insertAuditAndBump(ctx, queries, user, "Kosongkan data workspace", detail, "", "")
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return pkgErr.ErrFailCommitTx
	}
	s.publishState(user.WorkspaceID, version)
	return nil
}
