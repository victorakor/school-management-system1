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

// StudentsHandler handles student management, promotion, and transfer.
type StudentsHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewStudentsHandler creates a new StudentsHandler.
func NewStudentsHandler(db *gorm.DB, cfg *config.Config) *StudentsHandler {
	return &StudentsHandler{db: db, cfg: cfg}
}

// ListStudents handles GET /api/students
func (h *StudentsHandler) ListStudents(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	query := h.db.Where("school_id = ? AND is_archived = false", claims.SchoolID)

	// Division scope enforcement per RBAC spec:
	//   HEAD_TEACHER / ASST_HEAD_TEACHER — nursery+primary only (PermManagePupils)
	//   PRINCIPAL / VICE_PRINCIPAL       — secondary only (PermManageStudents, DivisionScope=SECONDARY)
	//   OWNER / EXAM_OFFICER             — unrestricted
	switch claims.Role {
	case models.RoleHeadTeacher, models.RoleAsstHeadTeacher:
		query = query.Joins("JOIN divisions d ON d.id = students.division_id").
			Where("d.scope IN ?", []string{"NURSERY", "PRIMARY"})
	case models.RolePrincipal, models.RoleVicePrincipal:
		query = query.Joins("JOIN divisions d ON d.id = students.division_id").
			Where("d.scope = ?", "SECONDARY")
	}

	if divID := r.URL.Query().Get("division_id"); divID != "" {
		query = query.Where("students.division_id = ?", divID)
	}
	if classID := r.URL.Query().Get("class_id"); classID != "" {
		// Join with StudentClassHistory to filter by current class
		query = query.Joins("JOIN student_class_histories sch ON sch.student_id = students.id").
			Where("sch.class_id = ?", classID)
	}
	if search := r.URL.Query().Get("search"); search != "" {
		query = query.Where("LOWER(students.full_name) LIKE LOWER(?)", "%"+search+"%")
	}
	if alumni := r.URL.Query().Get("alumni"); alumni == "true" {
		query = query.Where("students.is_alumni = true")
	} else {
		query = query.Where("students.is_alumni = false")
	}
	var students []models.Student
	query.Order("students.full_name ASC").Find(&students)
	utils.RespondSuccess(w, http.StatusOK, "", students)
}

// GetStudent handles GET /api/students/:id
func (h *StudentsHandler) GetStudent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}
	var student models.Student
	if err := h.db.First(&student, "id = ? AND is_archived = false", id).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Student not found")
		return
	}
	// Load class history
	var history []models.StudentClassHistory
	h.db.Where("student_id = ?", id).Order("created_at DESC").Find(&history)
	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"student": student,
		"history": history,
	})
}

// UpdateStudent handles PUT /api/students/:id
func (h *StudentsHandler) UpdateStudent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}
	var req struct {
		FullName  string `json:"full_name"`
		PhotoURL  string `json:"photo_url"`
		IsActive  *bool  `json:"is_active"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	updates := map[string]interface{}{}
	if req.FullName != "" {
		updates["full_name"] = req.FullName
	}
	if req.PhotoURL != "" {
		updates["photo_url"] = req.PhotoURL
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if err := h.db.Model(&models.Student{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update student")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Student updated", nil)
}

// PromoteStudents handles POST /api/students/promote — bulk or individual promotion
func (h *StudentsHandler) PromoteStudents(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	var req struct {
		SessionID string `json:"session_id"`
		TermID    string `json:"term_id"`
		Records   []struct {
			StudentID       string `json:"student_id"`
			NewClassID      string `json:"new_class_id"`
			Action          string `json:"action"` // "promote", "retain", "graduate"
			RetentionReason string `json:"retention_reason"`
		} `json:"records"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	v := validators.New()
	v.Required("session_id", req.SessionID, "Session")
	v.Required("term_id", req.TermID, "Term")
	if len(req.Records) == 0 {
		v.Custom("records", "At least one student record is required")
	}
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}
	sessionID, _ := uuid.Parse(req.SessionID)
	termID, _ := uuid.Parse(req.TermID)
	promoted, retained, graduated := 0, 0, 0
	for _, rec := range req.Records {
		studentID, err := uuid.Parse(rec.StudentID)
		if err != nil {
			continue
		}
		switch rec.Action {
		case "graduate":
			h.db.Model(&models.Student{}).Where("id = ?", studentID).
				Updates(map[string]interface{}{"is_alumni": true, "is_active": false})
			graduated++
		case "retain":
			history := &models.StudentClassHistory{
				StudentID:       studentID,
				SessionID:       sessionID,
				TermID:          termID,
				PromotedBy:      &claims.UserID,
				RetentionReason: rec.RetentionReason,
			}
			if rec.NewClassID != "" {
				cid, _ := uuid.Parse(rec.NewClassID)
				history.ClassID = cid
			}
			h.db.Create(history)
			retained++
		default: // promote
			if rec.NewClassID == "" {
				continue
			}
			newClassID, _ := uuid.Parse(rec.NewClassID)
			promoterID := claims.UserID
			history := &models.StudentClassHistory{
				StudentID:  studentID,
				ClassID:    newClassID,
				SessionID:  sessionID,
				TermID:     termID,
				PromotedBy: &promoterID,
			}
			h.db.Create(history)
			// Update student's current class
			h.db.Model(&models.Student{}).Where("id = ?", studentID).
				Update("division_id", h.getClassDivisionID(newClassID))
			promoted++
		}
	}
	utils.RespondSuccess(w, http.StatusOK, "Promotion complete", map[string]interface{}{
		"promoted":  promoted,
		"retained":  retained,
		"graduated": graduated,
	})
}

// ArchiveStudent handles DELETE /api/students/:id (archives, never deletes)
func (h *StudentsHandler) ArchiveStudent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}
	h.db.Model(&models.Student{}).Where("id = ?", id).
		Updates(map[string]interface{}{"is_archived": true, "is_active": false})
	utils.RespondSuccess(w, http.StatusOK, "Student archived", nil)
}

func (h *StudentsHandler) getClassDivisionID(classID uuid.UUID) uuid.UUID {
	var class models.Class
	h.db.First(&class, "id = ?", classID)
	return class.DivisionID
}
