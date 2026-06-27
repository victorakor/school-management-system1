package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/config"
	"school-platform/internal/jobs"
	"school-platform/internal/middleware"
	"school-platform/internal/models"
	"school-platform/internal/services"
	"school-platform/internal/utils"
)

// ResultsHandler handles result calculation, remarks, and publication.
type ResultsHandler struct {
	db        *gorm.DB
	cfg       *config.Config
	svc       *services.ResultService
	jobClient *jobs.Client
}

// NewResultsHandler creates a new ResultsHandler.
func NewResultsHandler(db *gorm.DB, cfg *config.Config, svc *services.ResultService, jobClient *jobs.Client) *ResultsHandler {
	return &ResultsHandler{db: db, cfg: cfg, svc: svc, jobClient: jobClient}
}

// CalculateResults handles POST /api/results/calculate
func (h *ResultsHandler) CalculateResults(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClassID   string `json:"class_id"`
		SessionID string `json:"session_id"`
		TermID    string `json:"term_id"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	classID, _ := uuid.Parse(req.ClassID)
	sessionID, _ := uuid.Parse(req.SessionID)
	termID, _ := uuid.Parse(req.TermID)

	if err := h.svc.CalculateResults(classID, sessionID, termID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Results calculated successfully", nil)
}

// GetResult handles GET /api/results/:id
func (h *ResultsHandler) GetResult(w http.ResponseWriter, r *http.Request) {
	resultID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid result ID")
		return
	}

	claims := middleware.GetClaims(r)

	var result models.Result
	if err := h.db.Preload("Student").Preload("Class").Preload("Session").Preload("Term").
		First(&result, "id = ?", resultID).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Result not found")
		return
	}

	// Students/parents can only see published results
	if (claims.Role == models.RoleStudent || claims.Role == models.RolePupil || claims.Role == models.RoleParent) &&
		!result.IsPublished {
		utils.RespondError(w, http.StatusForbidden, "Results have not been published yet")
		return
	}

	var subjects []models.ResultSubject
	h.db.Preload("Subject").Where("result_id = ?", resultID).Find(&subjects)

	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"result":   result,
		"subjects": subjects,
	})
}

// ListResults handles GET /api/results
func (h *ResultsHandler) ListResults(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var results []models.Result
	query := h.db.Preload("Student").Preload("Class")

	switch claims.Role {
	case models.RoleStudent, models.RolePupil:
		// Find student record for this user
		var student models.Student
		if err := h.db.Where("parent_id = ?", claims.UserID).First(&student).Error; err == nil {
			query = query.Where("student_id = ? AND is_published = true", student.ID)
		}
	case models.RoleParent:
		var students []models.Student
		h.db.Where("parent_id = ?", claims.UserID).Find(&students)
		ids := make([]uuid.UUID, len(students))
		for i, s := range students {
			ids[i] = s.ID
		}
		query = query.Where("student_id IN ? AND is_published = true", ids)
	default:
		// Admin roles — filter by class/session/term
		if classID := r.URL.Query().Get("class_id"); classID != "" {
			query = query.Where("class_id = ?", classID)
		}
		if sessionID := r.URL.Query().Get("session_id"); sessionID != "" {
			query = query.Where("session_id = ?", sessionID)
		}
		if termID := r.URL.Query().Get("term_id"); termID != "" {
			query = query.Where("term_id = ?", termID)
		}
	}

	query.Order("class_position ASC").Find(&results)
	utils.RespondSuccess(w, http.StatusOK, "", results)
}

// UpdateRemarks handles PUT /api/results/:id/remarks
func (h *ResultsHandler) UpdateRemarks(w http.ResponseWriter, r *http.Request) {
	resultID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid result ID")
		return
	}

	var req struct {
		ClassTeacherRemark string `json:"class_teacher_remark"`
		AdminRemark        string `json:"admin_remark"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if req.ClassTeacherRemark != "" {
		updates["class_teacher_remark"] = req.ClassTeacherRemark
	}
	if req.AdminRemark != "" {
		updates["admin_remark"] = req.AdminRemark
	}

	if err := h.db.Model(&models.Result{}).Where("id = ?", resultID).Updates(updates).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update remarks")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Remarks updated", nil)
}

// PublishResults handles POST /api/results/publish
func (h *ResultsHandler) PublishResults(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req struct {
		ClassID   string `json:"class_id"`
		SessionID string `json:"session_id"`
		TermID    string `json:"term_id"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	classID, _ := uuid.Parse(req.ClassID)
	sessionID, _ := uuid.Parse(req.SessionID)
	termID, _ := uuid.Parse(req.TermID)

	if err := h.svc.PublishResults(classID, sessionID, termID, claims.UserID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to publish results")
		return
	}

	// Enqueue report card PDF + notification for each published result
	if h.jobClient != nil {
		var publishedResults []models.Result
		h.db.Select("id, student_id").
			Where("class_id = ? AND session_id = ? AND term_id = ? AND is_published = true",
				classID, sessionID, termID).
			Find(&publishedResults)
		for _, res := range publishedResults {
			_ = h.jobClient.EnqueueReportCardPDF(res.ID.String(), claims.SchoolID.String())
			_ = h.jobClient.EnqueueNotification(
				res.StudentID.String(),
				"Results Published",
				"Your results are now available. Tap to view your report card.",
				"result_published",
				res.ID.String(),
			)
		}
	}

	utils.RespondSuccess(w, http.StatusOK, "Results published successfully", nil)
}
