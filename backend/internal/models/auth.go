package models

import "time"

type User struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	App         string `json:"app"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
}

type Session struct {
	User      User
	CSRF      string
	ExpiresAt time.Time
}

type AuthResult struct {
	Token     string
	CSRF      string
	ExpiresAt int64
	User      User
}

type RegisterRequest struct {
	Name      string `json:"name" validate:"required,min=2,max=100"`
	Email     string `json:"email" validate:"required,email,max=254"`
	Password  string `json:"password" validate:"required,min=12,max=72"`
	Workspace string `json:"workspace" validate:"required,min=2,max=100"`
	Role      string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email,max=254"`
	Password string `json:"password" validate:"required,max=72"`
}

type ChangePasswordRequest struct {
	Current  string `json:"current" validate:"required,max=72"`
	Password string `json:"password" validate:"required,min=12,max=72"`
}

type AuthResponse struct {
	User User   `json:"user"`
	CSRF string `json:"csrf"`
}

type ConfigResponse struct {
	Kind    string   `json:"kind"`
	Brand   string   `json:"brand"`
	Tagline string   `json:"tagline"`
	Roles   []string `json:"roles"`
}
