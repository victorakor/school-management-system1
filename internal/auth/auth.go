// Package auth handles JWT creation, parsing, and cookie management.
package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"school-platform/internal/config"
	"school-platform/internal/models"
)

const (
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"
)

// Claims holds the JWT payload for authenticated users.
type Claims struct {
	UserID        uuid.UUID             `json:"user_id"`
	SchoolID      uuid.UUID             `json:"school_id"`
	Role          models.Role           `json:"role"`
	DivisionScope models.DivisionScope  `json:"division_scope"`
	Email         string                `json:"email"`
	FullName      string                `json:"full_name"`
	jwt.RegisteredClaims
}

// TokenPair holds both access and refresh tokens.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// GenerateTokenPair creates a new access + refresh token pair for a user.
func GenerateTokenPair(user *models.User, cfg *config.Config) (*TokenPair, error) {
	accessToken, err := generateToken(user, cfg.JWTSecret, cfg.JWTAccessTTL)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to generate access token: %w", err)
	}

	refreshToken, err := generateToken(user, cfg.JWTRefreshSecret, cfg.JWTRefreshTTL)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func generateToken(user *models.User, secret string, ttl time.Duration) (string, error) {
	claims := &Claims{
		UserID:        user.ID,
		SchoolID:      user.SchoolID,
		Role:          user.Role,
		DivisionScope: user.DivisionScope,
		Email:         user.Email,
		FullName:      user.FullName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseAccessToken validates and parses an access token string.
func ParseAccessToken(tokenStr string, cfg *config.Config) (*Claims, error) {
	return parseToken(tokenStr, cfg.JWTSecret)
}

// ParseRefreshToken validates and parses a refresh token string.
func ParseRefreshToken(tokenStr string, cfg *config.Config) (*Claims, error) {
	return parseToken(tokenStr, cfg.JWTRefreshSecret)
}

func parseToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth: invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("auth: invalid token claims")
	}

	return claims, nil
}

// SetTokenCookies writes both tokens as HTTP-only cookies on the response.
func SetTokenCookies(w http.ResponseWriter, pair *TokenPair, cfg *config.Config) {
	secure := cfg.IsProduction()

	http.SetCookie(w, &http.Cookie{
		Name:     AccessTokenCookie,
		Value:    pair.AccessToken,
		Path:     "/",
		MaxAge:   int(cfg.JWTAccessTTL.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    pair.RefreshToken,
		Path:     "/",
		MaxAge:   int(cfg.JWTRefreshTTL.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearTokenCookies removes both auth cookies (logout).
func ClearTokenCookies(w http.ResponseWriter) {
	for _, name := range []string{AccessTokenCookie, RefreshTokenCookie} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		})
	}
}

// GetAccessTokenFromRequest extracts the access token from the request cookie.
func GetAccessTokenFromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie(AccessTokenCookie)
	if err != nil {
		return "", fmt.Errorf("auth: access token cookie not found: %w", err)
	}
	return cookie.Value, nil
}

// GetRefreshTokenFromRequest extracts the refresh token from the request cookie.
func GetRefreshTokenFromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie(RefreshTokenCookie)
	if err != nil {
		return "", fmt.Errorf("auth: refresh token cookie not found: %w", err)
	}
	return cookie.Value, nil
}
