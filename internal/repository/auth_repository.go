package repository

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/mohamedelhefni/siraaj/internal/domain"
)

type AuthRepository interface {
	CreateFirstUser(user domain.User) error
	HasAdministrator() (bool, error)
	CreateUser(user domain.User) error
	FindUserByEmail(email string) (domain.User, error)
	FindUserByID(id string) (domain.User, error)
	ListUsers() ([]domain.User, error)
	SetUserActive(id string, active bool) error
	EnsureProjectOwner(userID, projectID string) error
	ListProjects(userID string) ([]string, error)
	CreateTrackingToken(token domain.TrackingToken, tokenHash string) error
	ListTrackingTokens(userID string) ([]domain.TrackingToken, error)
	FindTrackingToken(tokenHash string) (domain.TrackingIdentity, error)
	TouchTrackingToken(id string, at time.Time) error
	RevokeTrackingToken(id, userID string) error
}

type authRepository struct{ db *sql.DB }

func NewAuthRepository(db *sql.DB) AuthRepository { return &authRepository{db: db} }

func (r *authRepository) CreateFirstUser(user domain.User) error {
	transaction, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	var guardID uint8
	if err := transaction.QueryRow(`INSERT INTO auth_bootstrap_guard (id) VALUES (1)
		ON CONFLICT DO NOTHING RETURNING id`).Scan(&guardID); errors.Is(err, sql.ErrNoRows) {
		return domain.ErrBootstrapComplete
	} else if err != nil {
		return err
	}
	if _, err := transaction.Exec(`INSERT INTO users (id, email, password_hash, role, active, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		user.ID, user.Email, user.PasswordHash, user.Role, user.Active, user.CreatedAt); err != nil {
		return err
	}
	if _, err := transaction.Exec(`INSERT INTO projects (id, owner_id, created_at)
		SELECT project_id, ?, ? FROM (
			SELECT DISTINCT project_id FROM events WHERE project_id IS NOT NULL AND project_id != ''
			UNION SELECT DISTINCT project_id FROM short_links WHERE project_id IS NOT NULL AND project_id != ''
		) legacy ON CONFLICT DO NOTHING`, user.ID, user.CreatedAt); err != nil {
		return err
	}
	return transaction.Commit()
}

func (r *authRepository) HasAdministrator() (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE role = 'admin' AND active = TRUE)").Scan(&exists)
	return exists, err
}

func (r *authRepository) CreateUser(user domain.User) error {
	_, err := r.db.Exec(`INSERT INTO users (id, email, password_hash, role, active, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		user.ID, user.Email, user.PasswordHash, user.Role, user.Active, user.CreatedAt)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return domain.ErrEmailTaken
	}
	return err
}

func (r *authRepository) FindUserByEmail(email string) (domain.User, error) {
	return r.findUser("SELECT id, email, password_hash, role, active, created_at FROM users WHERE email = ?", email)
}

func (r *authRepository) FindUserByID(id string) (domain.User, error) {
	return r.findUser("SELECT id, email, password_hash, role, active, created_at FROM users WHERE id = ?", id)
}

func (r *authRepository) findUser(query, value string) (domain.User, error) {
	var user domain.User
	err := r.db.QueryRow(query, value).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.Active, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrInvalidCredentials
	}
	return user, err
}

func (r *authRepository) ListUsers() ([]domain.User, error) {
	rows, err := r.db.Query("SELECT id, email, password_hash, role, active, created_at FROM users ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []domain.User{}
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.Active, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *authRepository) SetUserActive(id string, active bool) error {
	result, err := r.db.Exec("UPDATE users SET active = ? WHERE id = ?", active, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrInvalidCredentials
	}
	return nil
}

func (r *authRepository) EnsureProjectOwner(userID, projectID string) error {
	if _, err := r.db.Exec(`INSERT INTO projects (id, owner_id, created_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`,
		projectID, userID, time.Now().UTC()); err != nil {
		return err
	}
	var ownerID string
	if err := r.db.QueryRow("SELECT owner_id FROM projects WHERE id = ?", projectID).Scan(&ownerID); err != nil {
		return err
	}
	if ownerID != userID {
		return domain.ErrProjectUnavailable
	}
	return nil
}

func (r *authRepository) ListProjects(userID string) ([]string, error) {
	rows, err := r.db.Query("SELECT id FROM projects WHERE owner_id = ? ORDER BY created_at, id", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	projects := []string{}
	for rows.Next() {
		var projectID string
		if err := rows.Scan(&projectID); err != nil {
			return nil, err
		}
		projects = append(projects, projectID)
	}
	return projects, rows.Err()
}

func (r *authRepository) CreateTrackingToken(token domain.TrackingToken, tokenHash string) error {
	_, err := r.db.Exec(`INSERT INTO tracking_tokens
		(id, user_id, project_id, name, token_hash, token_prefix, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, token.ID, token.UserID, token.ProjectID, token.Name, tokenHash, token.Prefix, token.CreatedAt)
	return err
}

func (r *authRepository) ListTrackingTokens(userID string) ([]domain.TrackingToken, error) {
	return r.listTrackingTokens(`SELECT id, user_id, project_id, name, token_prefix, created_at, last_used_at, revoked_at
		FROM tracking_tokens WHERE user_id = ? ORDER BY created_at DESC`, userID)
}

func (r *authRepository) listTrackingTokens(query string, arguments ...any) ([]domain.TrackingToken, error) {
	rows, err := r.db.Query(query, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tokens := []domain.TrackingToken{}
	for rows.Next() {
		var token domain.TrackingToken
		var lastUsed, revoked sql.NullTime
		if err := rows.Scan(&token.ID, &token.UserID, &token.ProjectID, &token.Name, &token.Prefix, &token.CreatedAt, &lastUsed, &revoked); err != nil {
			return nil, err
		}
		if lastUsed.Valid {
			token.LastUsedAt = &lastUsed.Time
		}
		if revoked.Valid {
			token.RevokedAt = &revoked.Time
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (r *authRepository) FindTrackingToken(tokenHash string) (domain.TrackingIdentity, error) {
	var identity domain.TrackingIdentity
	var lastUsed sql.NullTime
	err := r.db.QueryRow(`SELECT t.id, t.user_id, t.project_id, t.last_used_at FROM tracking_tokens t
		JOIN users u ON u.id = t.user_id
		JOIN projects p ON p.id = t.project_id AND p.owner_id = t.user_id
		WHERE t.token_hash = ? AND t.revoked_at IS NULL AND u.active = TRUE`, tokenHash).
		Scan(&identity.TokenID, &identity.UserID, &identity.ProjectID, &lastUsed)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TrackingIdentity{}, domain.ErrInvalidToken
	}
	if lastUsed.Valid {
		identity.LastUsedAt = &lastUsed.Time
	}
	return identity, err
}

func (r *authRepository) TouchTrackingToken(id string, at time.Time) error {
	_, err := r.db.Exec("UPDATE tracking_tokens SET last_used_at = ? WHERE id = ?", at, id)
	return err
}

func (r *authRepository) RevokeTrackingToken(id, userID string) error {
	return r.revokeTrackingToken("UPDATE tracking_tokens SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL AND user_id = ?", id, userID)
}

func (r *authRepository) revokeTrackingToken(query, id string, arguments ...any) error {
	queryArguments := []any{time.Now().UTC(), id}
	queryArguments = append(queryArguments, arguments...)
	result, err := r.db.Exec(query, queryArguments...)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrTokenNotFound
	}
	return nil
}
