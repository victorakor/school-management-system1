package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/config"
	"school-platform/internal/jobs"
	"school-platform/internal/middleware"
	"school-platform/internal/models"
	"school-platform/internal/services"
	"school-platform/internal/utils"
	"school-platform/internal/validators"
)

// AdmissionsHandler handles all admissions-related HTTP endpoints.
type AdmissionsHandler struct {
	db        *gorm.DB
	cfg       *config.Config
	svc       *services.AdmissionService
	jobClient *jobs.Client
}

// NewAdmissionsHandler creates a new AdmissionsHandler.
func NewAdmissionsHandler(db *gorm.DB, cfg *config.Config, svc *services.AdmissionService, jobClient *jobs.Client) *AdmissionsHandler {
	return &AdmissionsHandler{db: db, cfg: cfg, svc: svc, jobClient: jobClient}
}

// ─── Admission Window ──────────────────────────────────────────────────────────

// GetAdmissionWindow handles GET /api/admissions/window
func (h *AdmissionsHandler) GetAdmissionWindow(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	var window models.AdmissionWindow
	err := h.db.Preload("Session").
		Where("school_id = ? AND is_active = true", claims.SchoolID).
		First(&window).Error
	if err != nil {
		utils.RespondJSON(w, http.StatusOK, map[string]interface{}{"window": nil, "is_open": false})
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"window":  window,
		"is_open": time.Now().After(window.OpenDate) && time.Now().Before(window.CloseDate),
	})
}

// CreateAdmissionWindow handles POST /api/admissions/window
func (h *AdmissionsHandler) CreateAdmissionWindow(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	var req struct {
		SessionID                  string            `json:"session_id"`
		Divisions                  []string          `json:"divisions"`
		OpenDate                   string            `json:"open_date"`
		CloseDate                  string            `json:"close_date"`
		MaxSlotsPerDivision        map[string]int    `json:"max_slots_per_division"`
		AppointmentCapacityPerSlot int               `json:"appointment_capacity_per_slot"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("session_id", req.SessionID, "Session")
	v.Required("open_date", req.OpenDate, "Open date")
	v.Date("open_date", req.OpenDate)
	v.Required("close_date", req.CloseDate, "Close date")
	v.Date("close_date", req.CloseDate)
	if len(req.Divisions) == 0 {
		v.Custom("divisions", "At least one division must be selected")
	}
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	sessionID, _ := uuid.Parse(req.SessionID)
	openDate, _ := time.Parse("2006-01-02", req.OpenDate)
	closeDate, _ := time.Parse("2006-01-02", req.CloseDate)

	divSlice := make(models.JSONSlice, len(req.Divisions))
	for i, d := range req.Divisions {
		divSlice[i] = d
	}

	maxSlots := make(models.JSONMap)
	for k, v := range req.MaxSlotsPerDivision {
		maxSlots[k] = v
	}

	cap := req.AppointmentCapacityPerSlot
	if cap == 0 {
		cap = 5
	}

	window := &models.AdmissionWindow{
		SchoolID:                   claims.SchoolID,
		SessionID:                  sessionID,
		Divisions:                  divSlice,
		OpenDate:                   openDate,
		CloseDate:                  closeDate,
		MaxSlotsPerDivision:        maxSlots,
		AppointmentCapacityPerSlot: cap,
		IsActive:                   true,
	}

	if err := h.db.Create(window).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create admission window")
		return
	}
	utils.RespondSuccess(w, http.StatusCreated, "Admission window created", window)
}

// ─── Applications ──────────────────────────────────────────────────────────────

// ListApplications handles GET /api/admissions/applications
func (h *AdmissionsHandler) ListApplications(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var apps []models.Application
	query := h.db.Preload("Parent").Where("school_id = ? AND is_archived = false", claims.SchoolID)

	if status := r.URL.Query().Get("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if div := r.URL.Query().Get("division"); div != "" {
		query = query.Where("division = ?", div)
	}

	// Scope by division for non-owner roles
	if claims.DivisionScope != models.DivisionAll {
		query = query.Where("division = ?", claims.DivisionScope)
	}

	query.Order("created_at DESC").Find(&apps)
	utils.RespondSuccess(w, http.StatusOK, "", apps)
}

// GetApplication handles GET /api/admissions/applications/:id
func (h *AdmissionsHandler) GetApplication(w http.ResponseWriter, r *http.Request) {
	appID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	var app models.Application
	if err := h.db.Preload("Parent").First(&app, "id = ?", appID).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Application not found")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "", app)
}

// SubmitApplication handles POST /api/admissions/applications (parent submits)
func (h *AdmissionsHandler) SubmitApplication(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req struct {
		ChildName             string `json:"child_name"`
		ChildDOB              string `json:"child_dob"`
		ChildGender           string `json:"child_gender"`
		Division              string `json:"division"`
		PassportURL           string `json:"passport_url"`
		BirthCertURL          string `json:"birth_cert_url"`
		PrevSchool            string `json:"prev_school"`
		PrevClass             string `json:"prev_class"`
		PrevReportURL         string `json:"prev_report_url"`
		EmergencyContactName  string `json:"emergency_contact_name"`
		EmergencyContactPhone string `json:"emergency_contact_phone"`
		HomeAddress           string `json:"home_address"`
		MedicalConditions     string `json:"medical_conditions"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("child_name", req.ChildName, "Child's full name")
	v.Required("child_dob", req.ChildDOB, "Child's date of birth")
	v.Date("child_dob", req.ChildDOB)
	v.Required("child_gender", req.ChildGender, "Child's gender")
	v.Required("division", req.Division, "Division")
	v.Enum("division", req.Division, []string{"NURSERY", "PRIMARY", "SECONDARY"}, "Division")
	v.Required("passport_url", req.PassportURL, "Passport photograph")
	v.CloudinaryURL("passport_url", req.PassportURL)
	v.Required("birth_cert_url", req.BirthCertURL, "Birth certificate")
	v.CloudinaryURL("birth_cert_url", req.BirthCertURL)
	v.Required("emergency_contact_name", req.EmergencyContactName, "Emergency contact name")
	v.Required("emergency_contact_phone", req.EmergencyContactPhone, "Emergency contact phone")
	v.Phone("emergency_contact_phone", req.EmergencyContactPhone)
	v.Required("home_address", req.HomeAddress, "Home address")
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	childDOB, _ := time.Parse("2006-01-02", req.ChildDOB)
	division := models.DivisionScope(req.Division)

	// Get school ID from parent's account
	var parent models.User
	if err := h.db.First(&parent, "id = ?", claims.UserID).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to fetch parent account")
		return
	}

	// Check admission window is open
	isOpen, window, err := h.svc.IsAdmissionWindowOpen(parent.SchoolID, division)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to check admission window")
		return
	}
	if !isOpen {
		utils.RespondError(w, http.StatusForbidden, "Admissions are currently closed for this division")
		return
	}

	// Duplicate detection
	if err := h.svc.CheckDuplicate(parent.SchoolID, req.ChildName, childDOB); err != nil {
		utils.RespondError(w, http.StatusConflict, err.Error())
		return
	}

	app := &models.Application{
		SchoolID:              parent.SchoolID,
		ParentID:              claims.UserID,
		ChildName:             req.ChildName,
		ChildDOB:              childDOB,
		ChildGender:           req.ChildGender,
		Division:              division,
		PassportURL:           req.PassportURL,
		BirthCertURL:          req.BirthCertURL,
		PrevSchool:            req.PrevSchool,
		PrevClass:             req.PrevClass,
		PrevReportURL:         req.PrevReportURL,
		EmergencyContactName:  req.EmergencyContactName,
		EmergencyContactPhone: req.EmergencyContactPhone,
		HomeAddress:           req.HomeAddress,
		MedicalConditions:     req.MedicalConditions,
	}

	if err := h.svc.CreateApplication(app); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to submit application")
		return
	}

	// Assign appointment slot
	if window != nil {
		_ = h.svc.AssignAppointmentSlot(app.ID, window)
	}

	// Enqueue appointment letter PDF generation job
	if h.jobClient != nil {
		_ = h.jobClient.EnqueueAppointmentLetterPDF(app.ID.String(), app.SchoolID.String())
	}

	utils.RespondSuccess(w, http.StatusCreated, "Application submitted successfully", map[string]interface{}{
		"application_id": app.ID,
		"ref_number":     app.RefNumber,
	})
}

// UpdateApplicationStatus handles PUT /api/admissions/applications/:id/status
func (h *AdmissionsHandler) UpdateApplicationStatus(w http.ResponseWriter, r *http.Request) {
	appID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	claims := middleware.GetClaims(r)
	var req struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Enum("status", req.Status, []string{"PENDING", "UNDER_REVIEW", "ACCEPTED", "DECLINED"}, "Status")
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	if err := h.svc.UpdateStatus(appID, models.ApplicationStatus(req.Status), req.Reason, claims.UserID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update status")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "Application status updated", nil)
}

// RescheduleAppointment handles POST /api/admissions/applications/:id/reschedule
func (h *AdmissionsHandler) RescheduleAppointment(w http.ResponseWriter, r *http.Request) {
	appID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	var app models.Application
	if err := h.db.First(&app, "id = ?", appID).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Application not found")
		return
	}

	isOpen, window, err := h.svc.IsAdmissionWindowOpen(app.SchoolID, app.Division)
	if err != nil || !isOpen || window == nil {
		utils.RespondError(w, http.StatusForbidden, "No active admission window for rescheduling")
		return
	}

	if err := h.svc.RescheduleAppointment(appID, window); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to reschedule appointment")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "Appointment rescheduled", nil)
}

// GetParentApplications handles GET /api/admissions/my-applications (parent view)
func (h *AdmissionsHandler) GetParentApplications(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var apps []models.Application
	h.db.Where("parent_id = ? AND is_archived = false", claims.UserID).
		Order("created_at DESC").Find(&apps)

	utils.RespondSuccess(w, http.StatusOK, "", apps)
}
