package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/config"
	"school-platform/internal/middleware"
	"school-platform/internal/models"
	"school-platform/internal/utils"
	"school-platform/internal/validators"
)

// FeedHandler handles activity feed endpoints.
type FeedHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewFeedHandler creates a new FeedHandler.
func NewFeedHandler(db *gorm.DB, cfg *config.Config) *FeedHandler {
	return &FeedHandler{db: db, cfg: cfg}
}

// ListPosts handles GET /api/feed — public endpoint, paginated
func (h *FeedHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize := 12
	offset := (page - 1) * pageSize

	query := h.db.Preload("Poster").Where("is_published = true AND is_archived = false")

	if div := r.URL.Query().Get("division"); div != "" {
		query = query.Where("division_tag = ?", div)
	}

	var total int64
	query.Model(&models.ActivityPost{}).Count(&total)

	var posts []models.ActivityPost
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&posts)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"data":        posts,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		"has_more":    int64(offset+pageSize) < total,
	})
}

// CreatePost handles POST /api/feed (admin roles only)
func (h *FeedHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req struct {
		Caption     string `json:"caption"`
		DivisionTag string `json:"division_tag"`
		MediaURLs   []struct {
			URL  string `json:"url"`
			Type string `json:"type"`
		} `json:"media_urls"`
		IsPublished bool `json:"is_published"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("division_tag", req.DivisionTag, "Division tag")
	v.Enum("division_tag", req.DivisionTag, []string{"NURSERY", "PRIMARY", "SECONDARY", "ALL"}, "Division tag")
	if len(req.MediaURLs) == 0 {
		v.Custom("media_urls", "At least one photo or video is required")
	}
	for i, m := range req.MediaURLs {
		v.CloudinaryURL("media_url_"+strconv.Itoa(i), m.URL)
		v.Enum("media_type_"+strconv.Itoa(i), m.Type, []string{"IMAGE", "VIDEO"}, "Media type")
	}
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	media := make(models.MediaItemSlice, len(req.MediaURLs))
	for i, m := range req.MediaURLs {
		media[i] = models.MediaItem{URL: m.URL, Type: models.MediaType(m.Type)}
	}

	post := &models.ActivityPost{
		SchoolID:    claims.SchoolID,
		PostedBy:    claims.UserID,
		Caption:     req.Caption,
		MediaURLs:   media,
		DivisionTag: models.DivisionScope(req.DivisionTag),
		IsPublished: req.IsPublished,
	}

	if err := h.db.Create(post).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create post")
		return
	}
	utils.RespondSuccess(w, http.StatusCreated, "Post created", post)
}

// UpdatePost handles PUT /api/feed/:id
func (h *FeedHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	var req struct {
		Caption     string `json:"caption"`
		IsPublished *bool  `json:"is_published"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if req.Caption != "" {
		updates["caption"] = req.Caption
	}
	if req.IsPublished != nil {
		updates["is_published"] = *req.IsPublished
	}

	if err := h.db.Model(&models.ActivityPost{}).Where("id = ?", postID).Updates(updates).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update post")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Post updated", nil)
}

// ArchivePost handles DELETE /api/feed/:id — archives, never deletes
func (h *FeedHandler) ArchivePost(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	if err := h.db.Model(&models.ActivityPost{}).Where("id = ?", postID).
		Updates(map[string]interface{}{
			"is_archived": true,
			"is_published": false,
		}).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to archive post")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Post archived", nil)
}
