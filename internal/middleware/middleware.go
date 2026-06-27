// Package middleware provides HTTP middleware for authentication, authorization,
// CSRF protection, rate limiting, security headers, and request logging.
package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"

	"school-platform/internal/auth"
	"school-platform/internal/config"
	"school-platform/internal/models"
	"school-platform/internal/permissions"
	"school-platform/internal/utils"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	ContextKeyClaims contextKey = "claims"
	ContextKeyCSRF   contextKey = "csrf_token"
)

// ─── Auth Middleware ───────────────────────────────────────────────────────────

// Authenticate validates the JWT access token from the cookie.
// On success, injects Claims into the request context.
// On failure, attempts to refresh using the refresh token.
func Authenticate(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := auth.GetAccessTokenFromRequest(r)
			if err != nil {
				respondUnauthorized(w, "Authentication required")
				return
			}

			claims, err := auth.ParseAccessToken(tokenStr, cfg)
			if err != nil {
				// Try refresh token
				refreshStr, refreshErr := auth.GetRefreshTokenFromRequest(r)
				if refreshErr != nil {
					respondUnauthorized(w, "Session expired — please log in again")
					return
				}

				refreshClaims, refreshErr := auth.ParseRefreshToken(refreshStr, cfg)
				if refreshErr != nil {
					auth.ClearTokenCookies(w)
					respondUnauthorized(w, "Session expired — please log in again")
					return
				}

				// Build a minimal user for token generation
				user := &models.User{
					Email:         refreshClaims.Email,
					FullName:      refreshClaims.FullName,
					Role:          refreshClaims.Role,
					DivisionScope: refreshClaims.DivisionScope,
				}
				user.ID = refreshClaims.UserID
				user.SchoolID = refreshClaims.SchoolID

				pair, tokenErr := auth.GenerateTokenPair(user, cfg)
				if tokenErr != nil {
					respondUnauthorized(w, "Failed to refresh session")
					return
				}

				auth.SetTokenCookies(w, pair, cfg)
				claims = refreshClaims
			}

			ctx := context.WithValue(r.Context(), ContextKeyClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole ensures the authenticated user has one of the specified roles.
func RequireRole(roles ...models.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil {
				respondUnauthorized(w, "Authentication required")
				return
			}

			for _, role := range roles {
				if claims.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			respondForbidden(w, "Insufficient permissions for this action")
		})
	}
}

// RequirePermission ensures the authenticated user has the specified permission.
func RequirePermission(perm permissions.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil {
				respondUnauthorized(w, "Authentication required")
				return
			}

			if !permissions.HasPermission(claims.Role, perm) {
				respondForbidden(w, "You do not have permission to perform this action")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireDivisionScope enforces that the user's division scope matches the requested division.
func RequireDivisionScope(division models.DivisionScope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil {
				respondUnauthorized(w, "Authentication required")
				return
			}

			if !permissions.CanAccessDivision(claims.DivisionScope, division) {
				respondForbidden(w, "Access denied: your account does not have access to this division")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetClaims extracts JWT claims from the request context.
// Returns nil if not authenticated.
func GetClaims(r *http.Request) *auth.Claims {
	claims, _ := r.Context().Value(ContextKeyClaims).(*auth.Claims)
	return claims
}

// ─── CSRF Middleware ───────────────────────────────────────────────────────────

// CSRFProtect issues and validates CSRF tokens.
// GET/HEAD/OPTIONS requests receive a token in a meta tag (injected via template).
// POST/PUT/PATCH/DELETE requests must include the token in X-CSRF-Token header.
func CSRFProtect(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Safe methods — generate and inject token
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				token := generateCSRFToken(cfg.CSRFSecret)
				ctx := context.WithValue(r.Context(), ContextKeyCSRF, token)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Unsafe methods — validate token
			token := r.Header.Get("X-CSRF-Token")
			if token == "" {
				token = r.FormValue("csrf_token")
			}

			if !validateCSRFToken(token, cfg.CSRFSecret) {
				respondJSON(w, http.StatusForbidden, map[string]string{
					"error": "Invalid or missing CSRF token",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func generateCSRFToken(secret string) string {
	timestamp := time.Now().Format("2006010215") // hourly rotation
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	return hex.EncodeToString(mac.Sum(nil))
}

func validateCSRFToken(token, secret string) bool {
	// Check current hour and previous hour (to handle boundary cases)
	for _, offset := range []int{0, -1} {
		t := time.Now().Add(time.Duration(offset) * time.Hour)
		timestamp := t.Format("2006010215")
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(timestamp))
		expected := hex.EncodeToString(mac.Sum(nil))
		if hmac.Equal([]byte(token), []byte(expected)) {
			return true
		}
	}
	return false
}

// GetCSRFToken extracts the CSRF token from the request context.
func GetCSRFToken(r *http.Request) string {
	token, _ := r.Context().Value(ContextKeyCSRF).(string)
	if token == "" {
		return generateCSRFToken(config.Get().CSRFSecret)
	}
	return token
}

// ─── Security Headers Middleware ───────────────────────────────────────────────

// SecurityHeaders sets HTTP security headers on every response.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline'; "+ // unsafe-inline needed for inline GSAP/Chart.js
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: https://res.cloudinary.com; "+
				"font-src 'self'; "+
				"connect-src 'self' https://api.cloudinary.com; "+
				"frame-ancestors 'none';",
		)
		next.ServeHTTP(w, r)
	})
}

// ─── Logging Middleware ────────────────────────────────────────────────────────

// RequestLogger logs each HTTP request with method, path, status, and duration.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			log.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.Status()).
				Dur("duration", time.Since(start)).
				Str("ip", realIP(r)).
				Msg("request")
		}()

		next.ServeHTTP(ww, r)
	})
}

// ─── Cache Control Middleware ──────────────────────────────────────────────────

// NoCache sets Cache-Control: no-store on API responses.
func NoCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// StaticCache sets long-lived cache headers for static assets.
func StaticCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only cache versioned assets (those with content hash in filename)
		if strings.Contains(r.URL.Path, "/static/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		next.ServeHTTP(w, r)
	})
}

// ─── Real IP Helper ────────────────────────────────────────────────────────────

// realIP extracts the real client IP, respecting Cloudflare's CF-Connecting-IP header.
func realIP(r *http.Request) string {
	// Cloudflare sets this header with the real client IP
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For can be a comma-separated list; take the first
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	return r.RemoteAddr
}

// RealIP returns the real client IP for a request (exported for use in handlers).
func RealIP(r *http.Request) string {
	return realIP(r)
}

// ─── Response Helpers ──────────────────────────────────────────────────────────

func respondUnauthorized(w http.ResponseWriter, msg string) {
	// Check if this is an API request (expects JSON) or a page request (expects redirect)
	respondJSON(w, http.StatusUnauthorized, map[string]string{"error": msg})
}

func respondForbidden(w http.ResponseWriter, msg string) {
	respondJSON(w, http.StatusForbidden, map[string]string{"error": msg})
}

func respondJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = utils.WriteJSON(w, body)
}
