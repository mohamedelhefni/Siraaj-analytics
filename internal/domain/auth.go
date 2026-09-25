package domain

import (
	"errors"
	"time"
)

var (
	ErrBootstrapComplete  = errors.New("the first administrator has already been created")
	ErrEmailTaken         = errors.New("email address is already in use")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrForbidden          = errors.New("you do not have permission to perform this action")
	ErrTokenNotFound      = errors.New("tracking token not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrProjectUnavailable = errors.New("project is unavailable")
	ErrSetupRequired      = errors.New("server administrator setup is required")
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
}

type Principal struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Issued int64  `json:"iat"`
	Expiry int64  `json:"exp"`
}

type TrackingIdentity struct {
	TokenID    string
	UserID     string
	ProjectID  string
	LastUsedAt *time.Time
}

type TrackingToken struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	ProjectID  string     `json:"project_id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

type IssuedTrackingToken struct {
	TrackingToken
	Token string `json:"token"`
}
