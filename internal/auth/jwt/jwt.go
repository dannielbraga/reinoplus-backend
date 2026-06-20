package jwt

import (
	"crypto/rsa"
	"fmt"
	"os"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/reinoplus/reinoplus/internal/domain"
)

const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
)

type Manager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	accessTTL  time.Duration
	refreshTTL time.Duration
}

type Claims struct {
	UserID string          `json:"uid"`
	Email  string          `json:"email"`
	Role   domain.UserRole `json:"role"`
	Type   string          `json:"typ"`
	gojwt.RegisteredClaims
}

func NewManager(privateKeyPath, publicKeyPath string, accessTTL, refreshTTL time.Duration) (*Manager, error) {
	privatePEM, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	publicPEM, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}

	privateKey, err := gojwt.ParseRSAPrivateKeyFromPEM(privatePEM)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	publicKey, err := gojwt.ParseRSAPublicKeyFromPEM(publicPEM)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	return &Manager{
		privateKey: privateKey,
		publicKey:  publicKey,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}, nil
}

func (m *Manager) GenerateAccessToken(user domain.User) (string, error) {
	return m.generateToken(user, tokenTypeAccess, m.accessTTL)
}

func (m *Manager) GenerateRefreshToken(user domain.User) (string, error) {
	return m.generateToken(user, tokenTypeRefresh, m.refreshTTL)
}

func (m *Manager) generateToken(user domain.User, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		Type:   tokenType,
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := gojwt.NewWithClaims(gojwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

func (m *Manager) ParseAccessToken(tokenString string) (Claims, error) {
	return m.parseToken(tokenString, tokenTypeAccess)
}

func (m *Manager) ParseRefreshToken(tokenString string) (string, error) {
	claims, err := m.parseToken(tokenString, tokenTypeRefresh)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func (m *Manager) parseToken(tokenString, expectedType string) (Claims, error) {
	token, err := gojwt.ParseWithClaims(tokenString, &Claims{}, func(token *gojwt.Token) (any, error) {
		if token.Method != gojwt.SigningMethodRS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.publicKey, nil
	})
	if err != nil {
		return Claims{}, domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid || claims.Type != expectedType {
		return Claims{}, domain.ErrInvalidToken
	}

	return *claims, nil
}
