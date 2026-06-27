package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/config"
	"school-platform/internal/middleware"
	"school-platform/internal/models"
	"school-platform/internal/utils"
	"school-platform/internal/validators"
)

// TimetableHandler handles timetable creation, versioning, and retrieval.
type TimetableHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewTimetableHandler creates a new TimetableHandler.
func NewTimetableHandler(db *gorm.DB, cfg *config.Config) *TimetableHandler {
	return &TimetableHandler{db: db, cfg: cfg}
}

// ListTimetables handles GET /api/timetables
func (h *TimetableHandler) ListTimetables(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	query := h.db.Where("school_id = ?", claims.SchoolID)
	if divID := r.URL.Query().Get("division_id"); divID != "" {
		query = query.Where("division_id = ?", divID)
	}
	if classID := r.URL.Query().Get("class_id"); classID != "" {
		query = query.Where("class_id = ?", classID)
	}
	if sessionID := r.URL.Query().Get("session_id"); sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if termID := r.URL.Query().Get("term_id"); termID != "" {
		query = query.Where("term_id = ?", termID)
	}
	if currentOnly := r.URL.Query().Get("current"); currentOnly == "true" {
		query = query.Where("is_current = true")
	}
	var timetables []models.Timetable
	query.Order("version DESC").Find(&timetables)
	utils.RespondSuccess(w, http.StatusOK, "", timetables)
}

// GetTimetable handles GET /api/timetables/:id
func (h *TimetableHandler) GetTimetable(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid timetable ID")
		return
	}
	var timetable models.Timetable
	if err := h.db.First(&timetable, "id = ?", id).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Timetable not found")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "", timetable)
}

// CreateBuilderTimetable handles POST /api/timetables/builder
func (h *TimetableHandler) CreateBuilderTimetable(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	var req struct {
		DivisionID    string           `json:"division_id"`
		ClassID       string           `json:"class_id"`
		SessionID     string           `json:"session_id"`
		TermID        string           `json:"term_id"`
		EffectiveFrom string           `json:"effective_from"`
		EffectiveTo   string           `json:"effective_to"`
		Data          models.JSONSlice `json:"data"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	v := validators.New()
	v.Required("division_id", req.DivisionID, "Division")
	v.Required("session_id", req.SessionID, "Session")
	v.Required("term_id", req.TermID, "Term")
	if len(req.Data) == 0 {
		v.Custom("data", "Timetable grid data is required")
	}
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}
	divisionID, _ := uuid.Parse(req.DivisionID)
	sessionID, _ := uuid.Parse(req.SessionID)
	termID, _ := uuid.Parse(req.TermID)
	var classIDPtr *uuid.UUID
	if req.ClassID != "" {
		cid, err := uuid.Parse(req.ClassID)
		if err == nil {
			classIDPtr = &cid
		}
	}
	effectiveFrom := time.Now()
	effectiveTo := time.Now().AddDate(0, 3, 0)
	if req.EffectiveFrom != "" {
		if t, err := time.Parse("2006-01-02", req.EffectiveFrom); err == nil {
			effectiveFrom = t
		}
	}
	if req.EffectiveTo != "" {
		if t, err := time.Parse("2006-01-02", req.EffectiveTo); err == nil {
			effectiveTo = t
		}
	}
	h.archiveCurrentTimetable(claims.SchoolID, divisionID, classIDPtr, sessionID, termID, effectiveFrom)
	var maxVersion int
	h.db.Model(&models.Timetable{}).
		Where("school_id = ? AND division_id = ? AND session_id = ? AND term_id = ?",
			claims.SchoolID, divisionID, sessionID, termID).
		Select("COALESCE(MAX(version), 0)").Scan(&maxVersion)
	timetable := &models.Timetable{
		SchoolID:      claims.SchoolID,
		DivisionID:    divisionID,
		ClassID:       classIDPtr,
		SessionID:     sessionID,
		TermID:        termID,
		Data:          models.JSONMap{"rows": req.Data},
		Version:       maxVersion + 1,
		Type:          models.TimetableBuilder,
		EffectiveFrom: effectiveFrom,
		EffectiveTo:   &effectiveTo,
		IsCurrent:     true,
		CreatedBy:     claims.UserID,
	}
	if err := h.db.Create(timetable).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to save timetable")
		return
	}
	utils.RespondSuccess(w, http.StatusCreated, "Timetable saved successfully", timetable)
}

// UploadTimetable handles POST /api/timetables/upload
func (h *TimetableHandler) UploadTimetable(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	var req struct {
		DivisionID    string `json:"division_id"`
		ClassID       string `json:"class_id"`
		SessionID     string `json:"session_id"`
		TermID        string `json:"term_id"`
		FileURL       string `json:"file_url"`
		FileType      string `json:"file_type"`
		EffectiveFrom string `json:"effective_from"`
		EffectiveTo   string `json:"effective_to"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	v := validators.New()
	v.Required("division_id", req.DivisionID, "Division")
	v.Required("session_id", req.SessionID, "Session")
	v.Required("term_id", req.TermID, "Term")
	v.Required("file_url", req.FileURL, "File URL")
	v.Required("file_type", req.FileType, "File type")
	if req.FileType != "pdf" && req.FileType != "xlsx" {
		v.Custom("file_type", "File type must be 'pdf' or 'xlsx'")
	}
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}
	divisionID, _ := uuid.Parse(req.DivisionID)
	sessionID, _ := uuid.Parse(req.SessionID)
	termID, _ := uuid.Parse(req.TermID)
	var classIDPtr *uuid.UUID
	if req.ClassID != "" {
		cid, _ := uuid.Parse(req.ClassID)
		classIDPtr = &cid
	}
	effectiveFrom := time.Now()
	effectiveTo := time.Now().AddDate(0, 3, 0)
	if req.EffectiveFrom != "" {
		if t, err := time.Parse("2006-01-02", req.EffectiveFrom); err == nil {
			effectiveFrom = t
		}
	}
	if req.EffectiveTo != "" {
		if t, err := time.Parse("2006-01-02", req.EffectiveTo); err == nil {
			effectiveTo = t
		}
	}
	h.archiveCurrentTimetable(claims.SchoolID, divisionID, classIDPtr, sessionID, termID, effectiveFrom)
	var maxVersion int
	h.db.Model(&models.Timetable{}).
		Where("school_id = ? AND division_id = ? AND session_id = ? AND term_id = ?",
			claims.SchoolID, divisionID, sessionID, termID).
		Select("COALESCE(MAX(version), 0)").Scan(&maxVersion)
	ttType := models.TimetableUploadedPDF
	if req.FileType == "xlsx" {
		ttType = models.TimetableUploadedXLSX
	}
	timetable := &models.Timetable{
		SchoolID:      claims.SchoolID,
		DivisionID:    divisionID,
		ClassID:       classIDPtr,
		SessionID:     sessionID,
		TermID:        termID,
		Data:          models.JSONMap{"file_url": req.FileURL},
		Version:       maxVersion + 1,
		Type:          ttType,
		EffectiveFrom: effectiveFrom,
		EffectiveTo:   &effectiveTo,
		IsCurrent:     true,
		CreatedBy:     claims.UserID,
	}
	if err := h.db.Create(timetable).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to save timetable")
		return
	}
	utils.RespondSuccess(w, http.StatusCreated, "Timetable uploaded successfully", timetable)
}

// GetTimetableHistory handles GET /api/timetables/history
func (h *TimetableHandler) GetTimetableHistory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	divisionID, _ := uuid.Parse(r.URL.Query().Get("division_id"))
	sessionID, _ := uuid.Parse(r.URL.Query().Get("session_id"))
	termID, _ := uuid.Parse(r.URL.Query().Get("term_id"))
	var timetables []models.Timetable
	h.db.Where("school_id = ? AND division_id = ? AND session_id = ? AND term_id = ?",
		claims.SchoolID, divisionID, sessionID, termID).
		Order("version DESC").Find(&timetables)
	utils.RespondSuccess(w, http.StatusOK, "", timetables)
}

// CheckTeacherConflict handles GET /api/timetables/check-conflict
func (h *TimetableHandler) CheckTeacherConflict(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	teacherID := r.URL.Query().Get("teacher_id")
	day := r.URL.Query().Get("day")
	periodStr := r.URL.Query().Get("period")
	sessionID := r.URL.Query().Get("session_id")
	termID := r.URL.Query().Get("term_id")
	if teacherID == "" || day == "" || periodStr == "" {
		utils.RespondError(w, http.StatusBadRequest, "teacher_id, day, and period are required")
		return
	}
	period, _ := strconv.Atoi(periodStr)
	var timetables []models.Timetable
	h.db.Where("school_id = ? AND session_id = ? AND term_id = ? AND is_current = true",
		claims.SchoolID, sessionID, termID).Find(&timetables)
	conflicts := []map[string]interface{}{}
	for _, tt := range timetables {
		for _, rowRaw := range tt.Data {
			row, ok := rowRaw.(map[string]interface{})
			if !ok {
				continue
			}
			rowPeriod, _ := row["period"].(float64)
			if int(rowPeriod) != period {
				continue
			}
			days, ok := row["days"].(map[string]interface{})
			if !ok {
				continue
			}
			slot, ok := days[day].(map[string]interface{})
			if !ok {
				continue
			}
			if slot["teacher_id"] == teacherID {
				conflicts = append(conflicts, map[string]interface{}{
					"timetable_id": tt.ID,
					"class_id":     tt.ClassID,
					"division_id":  tt.DivisionID,
					"day":          day,
					"period":       period,
					"subject":      slot["subject_name"],
				})
			}
		}
	}
	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"has_conflict": len(conflicts) > 0,
		"conflicts":    conflicts,
	})
}

func (h *TimetableHandler) archiveCurrentTimetable(
	schoolID, divisionID uuid.UUID,
	classID *uuid.UUID,
	sessionID, termID uuid.UUID,
	newEffectiveFrom time.Time,
) {
	query := h.db.Model(&models.Timetable{}).
		Where("school_id = ? AND division_id = ? AND session_id = ? AND term_id = ? AND is_current = true",
			schoolID, divisionID, sessionID, termID)
	if classID != nil {
		query = query.Where("class_id = ?", *classID)
	} else {
		query = query.Where("class_id IS NULL")
	}
	query.Updates(map[string]interface{}{
		"is_current":   false,
		"effective_to": newEffectiveFrom.AddDate(0, 0, -1),
	})
}
