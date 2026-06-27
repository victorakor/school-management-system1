package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/config"
	"school-platform/internal/middleware"
	"school-platform/internal/models"
	"school-platform/internal/utils"
	"school-platform/internal/validators"
)

// AnnouncementsHandler handles school announcements.
type AnnouncementsHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewAnnouncementsHandler creates a new AnnouncementsHandler.
func NewAnnouncementsHandler(db *gorm.DB, cfg *config.Config) *AnnouncementsHandler {
	return &AnnouncementsHandler{db: db, cfg: cfg}
}

// ListAnnouncements handles GET /api/announcements
func (h *AnnouncementsHandler) ListAnnouncements(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	query := h.db.Where("school_id = ? AND is_archived = false", claims.SchoolID)
	if div := r.URL.Query().Get("division"); div != "" {
		query = query.Where("target_division = ? OR target_division = 'ALL'", div)
	}
	if published := r.URL.Query().Get("published"); published == "true" {
		query = query.Where("is_published = true")
	}
	var announcements []models.Announcement
	query.Order("created_at DESC").Find(&announcements)
	utils.RespondSuccess(w, http.StatusOK, "", announcements)
}

// CreateAnnouncement handles POST /api/announcements
func (h *AnnouncementsHandler) CreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	var req struct {
		Title          string               `json:"title"`
		Body           string               `json:"body"`
		TargetDivision models.DivisionScope `json:"target_division"`
		IsPublished    bool                 `json:"is_published"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	v := validators.New()
	v.Required("title", req.Title, "Title")
	v.Required("body", req.Body, "Body")
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}
	div := req.TargetDivision
	if div == "" {
		div = models.DivisionAll
	}
	ann := &models.Announcement{
		SchoolID:       claims.SchoolID,
		PostedBy:       claims.UserID,
		Title:          req.Title,
		Body:           req.Body,
		TargetDivision: div,
		IsPublished:    req.IsPublished,
	}
	if err := h.db.Create(ann).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create announcement")
		return
	}
	utils.RespondSuccess(w, http.StatusCreated, "Announcement created", ann)
}

// UpdateAnnouncement handles PUT /api/announcements/:id
func (h *AnnouncementsHandler) UpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid announcement ID")
		return
	}
	var req struct {
		Title       string `json:"title"`
		Body        string `json:"body"`
		IsPublished *bool  `json:"is_published"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	updates := map[string]interface{}{}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Body != "" {
		updates["body"] = req.Body
	}
	if req.IsPublished != nil {
		updates["is_published"] = *req.IsPublished
	}
	h.db.Model(&models.Announcement{}).Where("id = ?", id).Updates(updates)
	utils.RespondSuccess(w, http.StatusOK, "Announcement updated", nil)
}

// ArchiveAnnouncement handles DELETE /api/announcements/:id
func (h *AnnouncementsHandler) ArchiveAnnouncement(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid announcement ID")
		return
	}
	h.db.Model(&models.Announcement{}).Where("id = ?", id).
		Updates(map[string]interface{}{"is_archived": true, "is_published": false})
	utils.RespondSuccess(w, http.StatusOK, "Announcement archived", nil)
}
