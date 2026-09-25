package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mohamedelhefni/siraaj/internal/domain"
	"github.com/mohamedelhefni/siraaj/internal/repository"
)

const passwordIterations = 210_000
const dummyPasswordHash = "pbkdf2_sha256$210000$AAAAAAAAAAAAAAAAAAAAAA$EVoeIgrsVBrgqWUfVwdzKFpYQm/CQ5jZk1pClNqHQg0"

var projectIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type AuthService interface {
	Bootstrap(email, password string) (domain.User, error)
	Signup(email, password string) (domain.User, error)
	Login(email, password string) (string, domain.User, error)
	AuthenticateAccessToken(token string) (domain.Principal, error)
	CreateUser(principal domain.Principal, email, password, role string) (domain.User, error)
	ListUsers(principal domain.Principal) ([]domain.User, error)
	SetUserActive(principal domain.Principal, id string, active bool) error
	CreateTrackingToken(principal domain.Principal, projectID, name string) (domain.IssuedTrackingToken, error)
	ListTrackingTokens(principal domain.Principal) ([]domain.TrackingToken, error)
	RevokeTrackingToken(principal domain.Principal, id string) error
	ListProjects(principal domain.Principal) ([]string, error)
	AuthenticateTrackingToken(token string) (domain.TrackingIdentity, error)
}

func (s *authService) Signup(email, password string) (domain.User, error) {
	ready, err := s.repository.HasAdministrator()
	if err != nil {
		return domain.User{}, err
	}
	if !ready {
		return domain.User{}, domain.ErrSetupRequired
	}
	user, err := s.newUser(email, password, "user")
	if err != nil {
		return domain.User{}, err
	}
	if err := s.repository.CreateUser(user); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

type authService struct {
	repository repository.AuthRepository
	secret     []byte
	tokenTTL   time.Duration
}

func NewAuthService(repository repository.AuthRepository, secret []byte, tokenTTL time.Duration) AuthService {
	return &authService{repository: repository, secret: secret, tokenTTL: tokenTTL}
}

func (s *authService) Bootstrap(email, password string) (domain.User, error) {
	user, err := s.newUser(email, password, "admin")
	if err != nil {
		return domain.User{}, err
	}
	if err := s.repository.CreateFirstUser(user); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (s *authService) Login(email, password string) (string, domain.User, error) {
	user, err := s.repository.FindUserByEmail(strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		verifyPassword(password, dummyPasswordHash)
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			return "", domain.User{}, err
		}
		return "", domain.User{}, domain.ErrInvalidCredentials
	}
	passwordValid := verifyPassword(password, user.PasswordHash)
	if !passwordValid || !user.Active {
		return "", domain.User{}, domain.ErrInvalidCredentials
	}
	now := time.Now().UTC()
	principal := domain.Principal{UserID: user.ID, Email: user.Email, Role: user.Role, Issued: now.Unix(), Expiry: now.Add(s.tokenTTL).Unix()}
	token, err := s.signPrincipal(principal)
	return token, user, err
}

func (s *authService) AuthenticateAccessToken(token string) (domain.Principal, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return domain.Principal{}, domain.ErrInvalidToken
	}
	unsigned := parts[0] + "." + parts[1]
	expected := hmac.New(sha256.New, s.secret)
	expected.Write([]byte(unsigned))
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(signature, expected.Sum(nil)) {
		return domain.Principal{}, domain.ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return domain.Principal{}, domain.ErrInvalidToken
	}
	var principal domain.Principal
	if err := json.Unmarshal(payload, &principal); err != nil || principal.UserID == "" || time.Now().Unix() >= principal.Expiry {
		return domain.Principal{}, domain.ErrInvalidToken
	}
	user, err := s.repository.FindUserByID(principal.UserID)
	if err != nil || !user.Active || user.Email != principal.Email || user.Role != principal.Role {
		return domain.Principal{}, domain.ErrInvalidToken
	}
	return principal, nil
}

func (s *authService) CreateUser(principal domain.Principal, email, password, role string) (domain.User, error) {
	if principal.Role != "admin" {
		return domain.User{}, domain.ErrForbidden
	}
	if role == "" {
		role = "user"
	}
	if role != "user" && role != "admin" {
		return domain.User{}, fmt.Errorf("%w: role must be user or admin", domain.ErrInvalidInput)
	}
	user, err := s.newUser(email, password, role)
	if err != nil {
		return domain.User{}, err
	}
	if err := s.repository.CreateUser(user); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (s *authService) newUser(email, password, role string) (domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) > 254 {
		return domain.User{}, fmt.Errorf("%w: a valid email address is required", domain.ErrInvalidInput)
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return domain.User{}, fmt.Errorf("%w: a valid email address is required", domain.ErrInvalidInput)
	}
	if len(password) < 10 || len(password) > 128 {
		return domain.User{}, fmt.Errorf("%w: password must be between 10 and 128 characters", domain.ErrInvalidInput)
	}
	hash, err := hashPassword(password)
	if err != nil {
		return domain.User{}, err
	}
	id, err := randomID()
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{ID: id, Email: email, PasswordHash: hash, Role: role, Active: true, CreatedAt: time.Now().UTC()}, nil
}

func (s *authService) ListUsers(principal domain.Principal) ([]domain.User, error) {
	if principal.Role != "admin" {
		return nil, domain.ErrForbidden
	}
	return s.repository.ListUsers()
}

func (s *authService) SetUserActive(principal domain.Principal, id string, active bool) error {
	if principal.Role != "admin" || (id == principal.UserID && !active) {
		return domain.ErrForbidden
	}
	return s.repository.SetUserActive(id, active)
}

func (s *authService) CreateTrackingToken(principal domain.Principal, projectID, name string) (domain.IssuedTrackingToken, error) {
	projectID = strings.TrimSpace(projectID)
	name = strings.TrimSpace(name)
	if !projectIDPattern.MatchString(projectID) {
		return domain.IssuedTrackingToken{}, fmt.Errorf("%w: project_id must use letters, numbers, dots, underscores, or hyphens", domain.ErrInvalidInput)
	}
	if name == "" || len(name) > 100 {
		return domain.IssuedTrackingToken{}, fmt.Errorf("%w: name is required and must be at most 100 characters", domain.ErrInvalidInput)
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return domain.IssuedTrackingToken{}, err
	}
	plain := "siraaj_trk_" + base64.RawURLEncoding.EncodeToString(raw)
	id, err := randomID()
	if err != nil {
		return domain.IssuedTrackingToken{}, err
	}
	token := domain.TrackingToken{ID: id, UserID: principal.UserID, ProjectID: projectID, Name: name, Prefix: plain[:22], CreatedAt: time.Now().UTC()}
	if err := s.repository.EnsureProjectOwner(principal.UserID, projectID); err != nil {
		return domain.IssuedTrackingToken{}, err
	}
	if err := s.repository.CreateTrackingToken(token, trackingTokenHash(plain)); err != nil {
		return domain.IssuedTrackingToken{}, err
	}
	return domain.IssuedTrackingToken{TrackingToken: token, Token: plain}, nil
}

func (s *authService) ListTrackingTokens(principal domain.Principal) ([]domain.TrackingToken, error) {
	return s.repository.ListTrackingTokens(principal.UserID)
}

func (s *authService) RevokeTrackingToken(principal domain.Principal, id string) error {
	return s.repository.RevokeTrackingToken(id, principal.UserID)
}

func (s *authService) ListProjects(principal domain.Principal) ([]string, error) {
	return s.repository.ListProjects(principal.UserID)
}

func (s *authService) AuthenticateTrackingToken(token string) (domain.TrackingIdentity, error) {
	if !strings.HasPrefix(token, "siraaj_trk_") {
		return domain.TrackingIdentity{}, domain.ErrInvalidToken
	}
	identity, err := s.repository.FindTrackingToken(trackingTokenHash(token))
	if err != nil {
		return domain.TrackingIdentity{}, err
	}
	now := time.Now().UTC()
	if identity.LastUsedAt == nil || now.Sub(*identity.LastUsedAt) >= 5*time.Minute {
		if err := s.repository.TouchTrackingToken(identity.TokenID, now); err != nil {
			return domain.TrackingIdentity{}, err
		}
	}
	return identity, nil
}

func (s *authService) signPrincipal(principal domain.Principal) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(principal)
	if err != nil {
		return "", err
	}
	unsigned := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	signature := hmac.New(sha256.New, s.secret)
	signature.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature.Sum(nil)), nil
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	derived := pbkdf2SHA256([]byte(password), salt, passwordIterations, 32)
	return fmt.Sprintf("pbkdf2_sha256$%d$%s$%s", passwordIterations,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(derived)), nil
}

func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2_sha256" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	salt, saltErr := base64.RawStdEncoding.DecodeString(parts[2])
	want, hashErr := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || saltErr != nil || hashErr != nil || iterations < 1 || len(want) == 0 {
		return false
	}
	got := pbkdf2SHA256([]byte(password), salt, iterations, len(want))
	return hmac.Equal(got, want)
}

func pbkdf2SHA256(password, salt []byte, iterations, keyLength int) []byte {
	result := make([]byte, 0, keyLength)
	for block := uint32(1); len(result) < keyLength; block++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		result = append(result, t...)
	}
	return result[:keyLength]
}

func randomID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func trackingTokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
