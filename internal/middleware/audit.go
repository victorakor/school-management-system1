package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/models"
)

// AuditAction holds the metadata for a single audit log entry.
type AuditAction struct {
	Action     string
	EntityType string
	EntityID   string
	Metadata   map[string]interface{}
}

// statusRecorder wraps ResponseWriter to capture the HTTP status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

// Audit returns middleware that writes an AuditLog row after the handler
// returns, but only on 2xx responses. The action and entityType are fixed
// per-route; the actor is derived from JWT claims.
//
// Usage:
//
//	r.With(middleware.Audit(db, "login", "user")).Post("/login", h.Login)
func Audit(db *gorm.DB, action, entityType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sr, r)

			// Only log on success (2xx)
			if sr.status < 200 || sr.status >= 300 {
				return
			}

			claims := GetClaims(r)
			if claims == nil {
				return
			}

			ip := r.Header.Get("CF-Connecting-IP")
			if ip == "" {
				ip = r.RemoteAddr
			}

			entry := &models.AuditLog{
				ID:         uuid.New(),
				UserID:     claims.UserID,
				Action:     action,
				EntityType: entityType,
				IPAddress:  ip,
				CreatedAt:  time.Now(),
			}

			_ = db.Create(entry).Error
		})
	}
}

// Record writes a single AuditLog entry inline (not as middleware).
// Use this inside handlers where the entity ID is known after the operation.
func Record(db *gorm.DB, r *http.Request, action AuditAction) {
	claims := GetClaims(r)
	if claims == nil {
		return
	}

	ip := r.Header.Get("CF-Connecting-IP")
	if ip == "" {
		ip = r.RemoteAddr
	}

	meta := models.JSONMap{}
	if action.Metadata != nil {
		for k, v := range action.Metadata {
			meta[k] = v
		}
	}

	entry := &models.AuditLog{
		ID:         uuid.New(),
		UserID:     claims.UserID,
		Action:     action.Action,
		EntityType: action.EntityType,
		EntityID:   action.EntityID,
		IPAddress:  ip,
		Metadata:   meta,
		CreatedAt:  time.Now(),
	}

	_ = db.Create(entry).Error
}
