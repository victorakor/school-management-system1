package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"

	"school-platform/internal/middleware"
	"school-platform/internal/models"
	"school-platform/internal/utils"
)

// AuditHandler handles audit log endpoints (Owner only).
type AuditHandler struct {
	db *gorm.DB
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(db *gorm.DB) *AuditHandler {
	return &AuditHandler{db: db}
}

// auditLogResponse is the sanitised shape returned to the client.
type auditLogResponse struct {
	ID         string         `json:"id"`
	UserID     string         `json:"user_id"`
	UserName   string         `json:"user_name"`
	UserRole   string         `json:"user_role"`
	Action     string         `json:"action"`
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	IPAddress  string         `json:"ip_address"`
	Metadata   models.JSONMap `json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
}

// ListAuditLogs handles GET /api/audit/logs
// Supports: page, limit, action, entity_type, user_id, from (YYYY-MM-DD), to (YYYY-MM-DD)
func (h *AuditHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	// Build the base query scoped to this school via the users join.
	query := h.db.Table("audit_logs al").
		Select(`al.id, al.user_id, u.full_name AS user_name, u.role AS user_role,
		        al.action, al.entity_type, al.entity_id,
		        al.ip_address, al.metadata, al.created_at`).
		Joins("JOIN users u ON u.id = al.user_id").
		Where("u.school_id = ?", claims.SchoolID)

	if action := r.URL.Query().Get("action"); action != "" {
		query = query.Where("al.action = ?", action)
	}
	if entityType := r.URL.Query().Get("entity_type"); entityType != "" {
		query = query.Where("al.entity_type = ?", entityType)
	}
	if userID := r.URL.Query().Get("user_id"); userID != "" {
		query = query.Where("al.user_id::text = ?", userID)
	}
	if from := r.URL.Query().Get("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			query = query.Where("al.created_at >= ?", t)
		}
	}
	if to := r.URL.Query().Get("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			query = query.Where("al.created_at < ?", t.Add(24*time.Hour))
		}
	}

	// Count before pagination
	var total int64
	query.Count(&total)

	var logs []auditLogResponse
	query.Order("al.created_at DESC").Limit(limit).Offset(offset).Scan(&logs)
	if logs == nil {
		logs = []auditLogResponse{}
	}

	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"logs":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// ExportAuditLogs handles GET /api/audit/logs/export
// Streams a CSV download of up to 10 000 most recent audit events for this school.
func (h *AuditHandler) ExportAuditLogs(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var logs []auditLogResponse
	h.db.Table("audit_logs al").
		Select(`al.id, al.user_id, u.full_name AS user_name, u.role AS user_role,
		        al.action, al.entity_type, al.entity_id,
		        al.ip_address, al.created_at`).
		Joins("JOIN users u ON u.id = al.user_id").
		Where("u.school_id = ?", claims.SchoolID).
		Order("al.created_at DESC").
		Limit(10000).
		Scan(&logs)

	filename := fmt.Sprintf("audit-logs-%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"ID", "User", "Role", "Action", "Entity Type", "Entity ID", "IP Address", "Timestamp"})
	for _, l := range logs {
		_ = cw.Write([]string{
			l.ID, l.UserName, l.UserRole, l.Action,
			l.EntityType, l.EntityID, l.IPAddress,
			l.CreatedAt.Format(time.RFC3339),
		})
	}
	cw.Flush()
}
