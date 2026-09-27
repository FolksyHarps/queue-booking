package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid or expired token")

// TokenManager issues and verifies HS256 JWTs and keeps an in-memory
// denylist of revoked token IDs (jti) until they would have expired anyway.
type TokenManager struct {
	secret []byte
	ttl    time.Duration

	mu      sync.Mutex
	revoked map[string]time.Time // jti -> expiry
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl, revoked: map[string]time.Time{}}
}

func (m *TokenManager) TTL() time.Duration { return m.ttl }

func (m *TokenManager) Issue(userID uint, role string) (string, error) {
	jti := make([]byte, 16)
	if _, err := rand.Read(jti); err != nil {
		return "", err
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  strconv.FormatUint(uint64(userID), 10),
		"role": role,
		"jti":  hex.EncodeToString(jti),
		"iat":  now.Unix(),
		"exp":  now.Add(m.ttl).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse validates the token and returns the user id, jti and expiry.
func (m *TokenManager) Parse(tokenStr string) (userID uint, jti string, exp time.Time, err error) {
	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return 0, "", time.Time{}, ErrInvalidToken
	}
	sub, _ := claims.GetSubject()
	id, perr := strconv.ParseUint(sub, 10, 64)
	jti, _ = claims["jti"].(string)
	expAt, _ := claims.GetExpirationTime()
	if perr != nil || jti == "" || expAt == nil {
		return 0, "", time.Time{}, ErrInvalidToken
	}
	if m.isRevoked(jti) {
		return 0, "", time.Time{}, ErrInvalidToken
	}
	return uint(id), jti, expAt.Time, nil
}

func (m *TokenManager) Revoke(jti string, exp time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for k, e := range m.revoked { // drop entries that expired naturally
		if e.Before(now) {
			delete(m.revoked, k)
		}
	}
	m.revoked[jti] = exp
}

func (m *TokenManager) isRevoked(jti string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.revoked[jti]
	return ok
}
