package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
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

func (s *Service) GetSession(ctx context.Context, token string) (models.Session, error) {
	if s.db == nil {
		return models.Session{}, pkgErr.New(pkgErr.ErrServiceUnavailable, "Penyimpanan sesi belum tersedia.")
	}
	if strings.TrimSpace(token) == "" {
		return models.Session{}, pkgErr.New(pkgErr.ErrUnauthorized, "Sesi berakhir. Masuk kembali untuk melanjutkan.")
	}

	row, err := sqlc.New(s.db.Pool()).GetSessionUser(ctx, sqlc.GetSessionUserParams{
		TokenHash: hashToken(token),
		App:       config.AppKindUMKM,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Session{}, pkgErr.New(pkgErr.ErrUnauthorized, "Sesi berakhir. Masuk kembali untuk melanjutkan.")
	}
	if err != nil {
		return models.Session{}, fmt.Errorf("get session: %w", err)
	}

	return models.Session{
		User: models.User{
			ID:          row.UserID,
			WorkspaceID: row.WorkspaceID,
			App:         row.App,
			Name:        row.Name,
			Email:       row.Email,
			Role:        row.Role,
		},
		CSRF:      row.Csrf,
		ExpiresAt: timeValue(row.ExpiresAt),
	}, nil
}

func (s *Service) Login(ctx context.Context, req models.LoginRequest) (models.AuthResult, error) {
	if s.db == nil {
		return models.AuthResult{}, pkgErr.New(pkgErr.ErrServiceUnavailable, "Database belum tersedia.")
	}

	email := normalizeEmail(req.Email)
	if !validEmail(email) || len(req.Password) > 72 {
		return models.AuthResult{}, pkgErr.New(pkgErr.ErrUnauthorized, "Email atau kata sandi belum cocok.")
	}

	row, err := sqlc.New(s.db.Pool()).GetUserForLogin(ctx, sqlc.GetUserForLoginParams{
		App:   config.AppKindUMKM,
		Email: email,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return models.AuthResult{}, pkgErr.New(pkgErr.ErrUnauthorized, "Email atau kata sandi belum cocok.")
	}
	if err != nil {
		return models.AuthResult{}, fmt.Errorf("get login user: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(req.Password)) != nil {
		return models.AuthResult{}, pkgErr.New(pkgErr.ErrUnauthorized, "Email atau kata sandi belum cocok.")
	}

	return s.createSession(ctx, models.User{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		App:         row.App,
		Name:        row.Name,
		Email:       row.Email,
		Role:        row.Role,
	})
}

func (s *Service) Register(ctx context.Context, req models.RegisterRequest) (models.AuthResult, error) {
	if s.db == nil {
		return models.AuthResult{}, pkgErr.New(pkgErr.ErrServiceUnavailable, "Database belum tersedia.")
	}

	name := strings.TrimSpace(req.Name)
	email := normalizeEmail(req.Email)
	workspace := strings.TrimSpace(req.Workspace)
	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = "owner"
	}
	if role != "owner" {
		return models.AuthResult{}, pkgErr.New(pkgErr.ErrInvalidRequest, "Aplikasi Temu Kas hanya menerima akun pemilik.")
	}
	if !validName(name) || !validEmail(email) || !passwordOK(req.Password) || !validName(workspace) || strings.ContainsRune(workspace, 0) {
		return models.AuthResult{}, pkgErr.New(pkgErr.ErrInvalidRequest, "Periksa nama, email, nama usaha, dan kata sandi 12-72 karakter.")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return models.AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	userID := newID()
	workspaceID := newID()
	code := strings.ToUpper(newID()[:10])

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return models.AuthResult{}, pkgErr.ErrFailInitializeTx
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := sqlc.New(tx)
	if _, err = queries.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{
		ID:   workspaceID,
		App:  config.AppKindUMKM,
		Name: workspace,
		Code: code,
	}); err != nil {
		return models.AuthResult{}, fmt.Errorf("create workspace: %w", err)
	}

	rows, err := queries.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           userID,
		WorkspaceID:  workspaceID,
		App:          config.AppKindUMKM,
		Name:         name,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         role,
	})
	if err != nil {
		return models.AuthResult{}, fmt.Errorf("create user: %w", err)
	}
	if rows == 0 {
		return models.AuthResult{}, pkgErr.New(pkgErr.ErrConflict, "Email sudah terdaftar. Gunakan halaman masuk.")
	}

	if err = tx.Commit(ctx); err != nil {
		return models.AuthResult{}, pkgErr.ErrFailCommitTx
	}

	return s.createSession(ctx, models.User{
		ID:          userID,
		WorkspaceID: workspaceID,
		App:         config.AppKindUMKM,
		Name:        name,
		Email:       email,
		Role:        role,
	})
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if s.db == nil {
		return pkgErr.New(pkgErr.ErrServiceUnavailable, "Database belum tersedia.")
	}
	if strings.TrimSpace(token) == "" {
		return nil
	}

	if _, err := sqlc.New(s.db.Pool()).DeleteSessionByTokenHash(ctx, hashToken(token)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, user models.User, req models.ChangePasswordRequest) error {
	if s.db == nil {
		return pkgErr.New(pkgErr.ErrServiceUnavailable, "Database belum tersedia.")
	}
	if !passwordOK(req.Password) {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "Kata sandi baru harus 12-72 karakter.")
	}

	queries := sqlc.New(s.db.Pool())
	currentHash, err := queries.GetUserPasswordHash(ctx, user.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return pkgErr.New(pkgErr.ErrUnauthorized, "Akun tidak ditemukan. Masuk kembali.")
	}
	if err != nil {
		return fmt.Errorf("get password hash: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.Current)) != nil {
		return pkgErr.New(pkgErr.ErrInvalidRequest, "Kata sandi saat ini belum cocok.")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return pkgErr.ErrFailInitializeTx
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txQueries := sqlc.New(tx)
	rows, err := txQueries.UpdateUserPasswordHash(ctx, sqlc.UpdateUserPasswordHashParams{
		UserID:       user.ID,
		PasswordHash: string(newHash),
	})
	if err != nil {
		return fmt.Errorf("update password hash: %w", err)
	}
	if rows == 0 {
		return pkgErr.New(pkgErr.ErrUnauthorized, "Akun tidak ditemukan. Masuk kembali.")
	}
	if _, err = txQueries.DeleteSessionsByUserID(ctx, user.ID); err != nil {
		return fmt.Errorf("delete user sessions: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return pkgErr.ErrFailCommitTx
	}

	return nil
}

func (s *Service) createSession(ctx context.Context, user models.User) (models.AuthResult, error) {
	token := newID() + newID()
	csrf := newID()
	expires := time.Now().Add(config.SessionTTL)

	if err := sqlc.New(s.db.Pool()).CreateSession(ctx, sqlc.CreateSessionParams{
		TokenHash: hashToken(token),
		UserID:    user.ID,
		Csrf:      csrf,
		ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true},
	}); err != nil {
		return models.AuthResult{}, fmt.Errorf("create session: %w", err)
	}

	return models.AuthResult{
		Token:     token,
		CSRF:      csrf,
		ExpiresAt: expires.Unix(),
		User:      user,
	}, nil
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validEmail(value string) bool {
	parsed, err := mail.ParseAddress(value)
	return err == nil && parsed.Address == value && len(value) < 255
}

func passwordOK(value string) bool {
	return len(value) >= 12 && len(value) <= 72
}

func validName(value string) bool {
	return len([]rune(value)) >= 2 && len([]rune(value)) <= 100 && !strings.ContainsRune(value, 0)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(value[:])
}
