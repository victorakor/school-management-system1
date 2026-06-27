package handlers

import (
	"net/http"
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

// AcademicHandler handles sessions, terms, classes, subjects, and teacher assignments.
type AcademicHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewAcademicHandler creates a new AcademicHandler.
func NewAcademicHandler(db *gorm.DB, cfg *config.Config) *AcademicHandler {
	return &AcademicHandler{db: db, cfg: cfg}
}

// ─── Academic Sessions ─────────────────────────────────────────────────────────

// ListSessions handles GET /api/sessions
func (h *AcademicHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	var sessions []models.AcademicSession
	h.db.Where("school_id = ? AND is_archived = false", claims.SchoolID).
		Order("start_date DESC").Find(&sessions)
	utils.RespondSuccess(w, http.StatusOK, "", sessions)
}

// CreateSession handles POST /api/sessions
func (h *AcademicHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req struct {
		Name      string `json:"name"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("name", req.Name, "Session name")
	v.Required("start_date", req.StartDate, "Start date")
	v.Date("start_date", req.StartDate)
	v.Required("end_date", req.EndDate, "End date")
	v.Date("end_date", req.EndDate)
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	session := &models.AcademicSession{
		SchoolID: claims.SchoolID,
		Name:     req.Name,
	}
	session.StartDate, _ = parseDate(req.StartDate)
	session.EndDate, _ = parseDate(req.EndDate)

	if err := h.db.Create(session).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	utils.RespondSuccess(w, http.StatusCreated, "Session created", session)
}

// SetActiveSession handles PUT /api/sessions/:id/activate
func (h *AcademicHandler) SetActiveSession(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	// Deactivate all sessions for this school
	h.db.Model(&models.AcademicSession{}).
		Where("school_id = ?", claims.SchoolID).
		Update("is_active", false)

	// Activate the selected session
	h.db.Model(&models.AcademicSession{}).
		Where("id = ? AND school_id = ?", sessionID, claims.SchoolID).
		Update("is_active", true)

	utils.RespondSuccess(w, http.StatusOK, "Session activated", nil)
}

// ─── Terms ─────────────────────────────────────────────────────────────────────

// ListTerms handles GET /api/sessions/:sessionId/terms
func (h *AcademicHandler) ListTerms(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionId"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	var terms []models.Term
	h.db.Where("session_id = ?", sessionID).Order("start_date ASC").Find(&terms)
	utils.RespondSuccess(w, http.StatusOK, "", terms)
}

// CreateTerm handles POST /api/sessions/:sessionId/terms
func (h *AcademicHandler) CreateTerm(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionId"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	var req struct {
		Name               string `json:"name"`
		StartDate          string `json:"start_date"`
		EndDate            string `json:"end_date"`
		NextResumptionDate string `json:"next_resumption_date"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("name", req.Name, "Term name")
	v.Required("start_date", req.StartDate, "Start date")
	v.Date("start_date", req.StartDate)
	v.Required("end_date", req.EndDate, "End date")
	v.Date("end_date", req.EndDate)
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	term := &models.Term{
		SessionID: sessionID,
		Name:      req.Name,
	}
	term.StartDate, _ = parseDate(req.StartDate)
	term.EndDate, _ = parseDate(req.EndDate)
	if req.NextResumptionDate != "" {
		term.NextResumptionDate, _ = parseDate(req.NextResumptionDate)
	}

	if err := h.db.Create(term).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create term")
		return
	}

	utils.RespondSuccess(w, http.StatusCreated, "Term created", term)
}

// ─── Classes ───────────────────────────────────────────────────────────────────

// ListClasses handles GET /api/classes
func (h *AcademicHandler) ListClasses(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var classes []models.Class
	query := h.db.Preload("Division").Preload("FormTeacher")

	// Filter by division if provided
	if divID := r.URL.Query().Get("division_id"); divID != "" {
		query = query.Where("division_id = ?", divID)
	} else {
		// Filter by user's division scope
		if claims.DivisionScope != models.DivisionAll {
			query = query.Joins("JOIN divisions ON divisions.id = classes.division_id").
				Where("divisions.name = ?", claims.DivisionScope)
		}
	}

	query.Order("name ASC").Find(&classes)
	utils.RespondSuccess(w, http.StatusOK, "", classes)
}

// CreateClass handles POST /api/classes
func (h *AcademicHandler) CreateClass(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DivisionID    string `json:"division_id"`
		Name          string `json:"name"`
		Stream        string `json:"stream"`
		FormTeacherID string `json:"form_teacher_id"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("division_id", req.DivisionID, "Division")
	v.Required("name", req.Name, "Class name")
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	divID, err := uuid.Parse(req.DivisionID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid division ID")
		return
	}

	class := &models.Class{
		DivisionID: divID,
		Name:       req.Name,
		Stream:     req.Stream,
	}

	if req.FormTeacherID != "" {
		ftID, err := uuid.Parse(req.FormTeacherID)
		if err == nil {
			class.FormTeacherID = &ftID
		}
	}

	if err := h.db.Create(class).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create class")
		return
	}

	utils.RespondSuccess(w, http.StatusCreated, "Class created", class)
}

// ─── Subjects ──────────────────────────────────────────────────────────────────

// ListSubjects handles GET /api/subjects
func (h *AcademicHandler) ListSubjects(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var subjects []models.Subject
	query := h.db.Preload("Division").Where("is_archived = false")

	if divID := r.URL.Query().Get("division_id"); divID != "" {
		query = query.Where("division_id = ?", divID)
	} else if claims.DivisionScope != models.DivisionAll {
		query = query.Joins("JOIN divisions ON divisions.id = subjects.division_id").
			Where("divisions.name = ?", claims.DivisionScope)
	}

	query.Order("name ASC").Find(&subjects)
	utils.RespondSuccess(w, http.StatusOK, "", subjects)
}

// CreateSubject handles POST /api/subjects
func (h *AcademicHandler) CreateSubject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DivisionID string `json:"division_id"`
		Name       string `json:"name"`
		Code       string `json:"code"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("division_id", req.DivisionID, "Division")
	v.Required("name", req.Name, "Subject name")
	v.Required("code", req.Code, "Subject code")
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	divID, err := uuid.Parse(req.DivisionID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid division ID")
		return
	}

	subject := &models.Subject{
		DivisionID: divID,
		Name:       req.Name,
		Code:       req.Code,
	}

	if err := h.db.Create(subject).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create subject")
		return
	}

	utils.RespondSuccess(w, http.StatusCreated, "Subject created", subject)
}

// ─── Teacher Assignments ───────────────────────────────────────────────────────

// AssignTeacher handles POST /api/assignments
func (h *AcademicHandler) AssignTeacher(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeacherID string `json:"teacher_id"`
		ClassID   string `json:"class_id"`
		SubjectID string `json:"subject_id"`
		SessionID string `json:"session_id"`
		TermID    string `json:"term_id"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	teacherID, _ := uuid.Parse(req.TeacherID)
	classID, _ := uuid.Parse(req.ClassID)
	subjectID, _ := uuid.Parse(req.SubjectID)
	sessionID, _ := uuid.Parse(req.SessionID)
	termID, _ := uuid.Parse(req.TermID)

	assignment := &models.TeacherAssignment{
		TeacherID: teacherID,
		ClassID:   classID,
		SubjectID: subjectID,
		SessionID: sessionID,
		TermID:    termID,
	}

	if err := h.db.Create(assignment).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to assign teacher")
		return
	}

	utils.RespondSuccess(w, http.StatusCreated, "Teacher assigned successfully", assignment)
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
