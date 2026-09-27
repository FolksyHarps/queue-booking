package service

import (
	"errors"
	"strings"
	"time"

	"github.com/queue-booking/auth-service/internal/model"
	"github.com/queue-booking/auth-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountSuspended   = errors.New("account suspended")
	ErrUserNotFound       = errors.New("user not found")
)

type AuthService struct {
	users  *repository.UserRepository
	tokens *TokenManager
}

func NewAuthService(users *repository.UserRepository, tokens *TokenManager) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

func (s *AuthService) Register(name, email, password, role string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &model.User{
		Name:         name,
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: string(hash),
		Role:         role,
		Status:       model.StatusActive,
	}
	if err := s.users.Create(u); err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return u, nil
}

// Login returns a signed JWT for valid, active accounts.
func (s *AuthService) Login(email, password string) (string, error) {
	u, err := s.users.FindByEmail(strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, repository.ErrNotFound) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", ErrInvalidCredentials
	}
	if u.Status != model.StatusActive {
		return "", ErrAccountSuspended
	}
	return s.tokens.Issue(u.ID, u.Role)
}

func (s *AuthService) TokenTTL() time.Duration { return s.tokens.TTL() }

// Authenticate resolves a bearer token to an active user.
func (s *AuthService) Authenticate(token string) (*model.User, string, time.Time, error) {
	id, jti, exp, err := s.tokens.Parse(token)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	u, err := s.users.FindByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, "", time.Time{}, ErrInvalidToken
	}
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if u.Status != model.StatusActive {
		return nil, "", time.Time{}, ErrAccountSuspended
	}
	return u, jti, exp, nil
}

func (s *AuthService) Logout(jti string, exp time.Time) {
	s.tokens.Revoke(jti, exp)
}

// EnsureAdmin creates the bootstrap admin account if the email is not taken yet.
func (s *AuthService) EnsureAdmin(email, password string) error {
	if email == "" || password == "" {
		return nil
	}
	_, err := s.users.FindByEmail(strings.ToLower(email))
	if err == nil {
		return nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return err
	}
	_, err = s.Register("Administrator", email, password, model.RoleAdmin)
	if errors.Is(err, ErrEmailTaken) {
		return nil
	}
	return err
}
