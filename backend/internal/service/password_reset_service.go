package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/repository/rekonsiliasi_repository_sqlc"
	pkgErr "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

const defaultPasswordResetTTL = 15 * time.Minute

func (s *Service) ForgotPassword(ctx context.Context, req models.ForgotPasswordRequest) error {
	if s.db == nil {
		return pkgErr.New(pkgErr.ErrServiceUnavailable, "Database belum tersedia.")
	}
	if s.passwordResetMailer == nil {
		return pkgErr.New(pkgErr.ErrServiceUnavailable, "Layanan email reset belum dikonfigurasi.")
	}

	email := normalizeEmail(req.Email)
	if !validEmail(email) {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "Masukkan alamat email yang valid.")
	}

	queries := sqlc.New(s.db.Pool())
	userID, err := queries.GetUserIDByEmail(ctx, sqlc.GetUserIDByEmailParams{
		App:   config.AppKindUMKM,
		Email: email,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// Do not reveal whether an account exists.
		return nil
	}
	if err != nil {
		return fmt.Errorf("find password reset user: %w", err)
	}

	token := newID() + newID()
	ttl := s.cfg.Email.ResetTTL
	if ttl <= 0 {
		ttl = defaultPasswordResetTTL
	}
	expiresAt := time.Now().Add(ttl)

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return pkgErr.ErrFailInitializeTx
	}
	defer func() { _ = tx.Rollback(ctx) }()
	txQueries := sqlc.New(tx)
	if err = txQueries.DeletePasswordResetTokensByUser(ctx, userID); err != nil {
		return fmt.Errorf("invalidate old password reset tokens: %w", err)
	}
	if err = txQueries.CreatePasswordResetToken(ctx, sqlc.CreatePasswordResetTokenParams{
		TokenHash: hashToken(token),
		UserID:    userID,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return pkgErr.ErrFailCommitTx
	}

	resetURL := s.passwordResetURL(token)
	if err := s.passwordResetMailer.SendPasswordReset(ctx, email, resetURL); err != nil {
		return fmt.Errorf("send password reset email: %w", err)
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, req models.ResetPasswordRequest) error {
	if s.db == nil {
		return pkgErr.New(pkgErr.ErrServiceUnavailable, "Database belum tersedia.")
	}
	if !validResetToken(req.Token) || !passwordOK(req.Password) {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "Tautan reset tidak valid atau sudah kedaluwarsa. Minta tautan baru.")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return pkgErr.ErrFailInitializeTx
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)
	row, err := queries.GetPasswordResetTokenForUpdate(ctx, hashToken(req.Token))
	if errors.Is(err, pgx.ErrNoRows) {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "Tautan reset tidak valid atau sudah kedaluwarsa. Minta tautan baru.")
	}
	if err != nil {
		return fmt.Errorf("get password reset token: %w", err)
	}
	if !row.ExpiresAt.Valid || !time.Now().Before(row.ExpiresAt.Time) {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "Tautan reset tidak valid atau sudah kedaluwarsa. Minta tautan baru.")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return fmt.Errorf("hash reset password: %w", err)
	}
	rows, err := queries.UpdateUserPasswordHash(ctx, sqlc.UpdateUserPasswordHashParams{
		UserID:       row.UserID,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		return fmt.Errorf("update reset password: %w", err)
	}
	if rows == 0 {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "Akun tidak ditemukan. Minta tautan baru.")
	}
	if _, err = queries.DeleteSessionsByUserID(ctx, row.UserID); err != nil {
		return fmt.Errorf("revoke sessions after reset: %w", err)
	}
	if err = queries.DeletePasswordResetTokensByUser(ctx, row.UserID); err != nil {
		return fmt.Errorf("consume password reset token: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return pkgErr.ErrFailCommitTx
	}
	return nil
}

func (s *Service) passwordResetURL(token string) string {
	base := strings.TrimRight(strings.TrimSpace(s.cfg.Email.AppURL), "/")
	if base == "" {
		base = "http://localhost:3000"
	}
	return base + "/app#reset?token=" + url.QueryEscape(token)
}

func validResetToken(token string) bool {
	if len(token) != 64 {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}
