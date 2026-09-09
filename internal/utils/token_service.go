package utils

import (
	"fmt"
	"time"

	"indagio-api/internal/config"
	"indagio-api/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type tokenService struct {
	accessSecret  string
	refreshSecret string
}

// NewTokenService creates a token service with secrets injected from config.
func NewTokenService(cfg config.AuthConfig) domain.TokenService {
	return &tokenService{
		accessSecret:  cfg.JWTSecret,
		refreshSecret: cfg.JWTRefreshSecret,
	}
}

func (s *tokenService) GenerateAccessToken(userID uuid.UUID, role domain.Role) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"role":    string(role),
		"exp":     time.Now().Add(3 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.accessSecret))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return tokenString, nil
}

func (s *tokenService) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.refreshSecret))
	if err != nil {
		return "", fmt.Errorf("sign refresh token: %w", err)
	}

	return tokenString, nil
}

func (s *tokenService) ValidateAccessToken(tokenStr string) (*domain.TokenClaims, error) {
	return s.parseToken(tokenStr, s.accessSecret)
}

func (s *tokenService) ValidateRefreshToken(tokenStr string) (*domain.TokenClaims, error) {
	return s.parseToken(tokenStr, s.refreshSecret)
}

func (s *tokenService) parseToken(tokenStr, secret string) (*domain.TokenClaims, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", domain.ErrUnauthorized)
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok || userIDStr == "" {
		return nil, fmt.Errorf("missing user_id claim: %w", domain.ErrUnauthorized)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id format: %w", domain.ErrUnauthorized)
	}

	roleStr, _ := claims["role"].(string)

	return &domain.TokenClaims{
		UserID: userID,
		Role:   domain.Role(roleStr),
	}, nil
}
