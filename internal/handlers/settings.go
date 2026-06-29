package handlers

import (
	"net/http"

	"gorm.io/gorm"

	"school-platform/internal/config"
	"school-platform/internal/middleware"
	"school-platform/internal/models"
	"school-platform/internal/utils"
	"school-platform/internal/validators"
)

// SettingsHandler handles school settings endpoints.
type SettingsHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewSettingsHandler creates a new SettingsHandler.
func NewSettingsHandler(db *gorm.DB, cfg *config.Config) *SettingsHandler {
	return &SettingsHandler{db: db, cfg: cfg}
}

// GetSchoolSettings handles GET /api/settings/school
func (h *SettingsHandler) GetSchoolSettings(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var school models.School
	if err := h.db.First(&school, "id = ?", claims.SchoolID).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "School not found")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "", school)
}

// UpdateSchoolSettings handles PUT /api/settings/school
func (h *SettingsHandler) UpdateSchoolSettings(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req struct {
		Name                   string `json:"name"`
		Motto                  string `json:"motto"`
		Address                string `json:"address"`
		Phone                  string `json:"phone"`
		Email                  string `json:"email"`
		LogoURL                string `json:"logo_url"`
		StampURL               string `json:"stamp_url"`
		SignaturePrincipalURL  string `json:"signature_principal_url"`
		SignatureHeadteacherURL string `json:"signature_headteacher_url"`
		PrimaryColor           string `json:"primary_color"`
		Prefix                 string `json:"prefix"`
		WatermarkEnabled       *bool  `json:"watermark_enabled"`
		BursarName             string `json:"bursar_name"`
		AdmissionDocumentsList string `json:"admission_documents_list"`
		SchoolDirections       string `json:"school_directions"`
		MaxVideoUploadMB       *int   `json:"max_video_upload_mb"`
		EstablishedYear        *int   `json:"established_year"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	if req.Email != "" {
		v.Email("email", req.Email)
	}
	if req.LogoURL != "" {
		v.CloudinaryURL("logo_url", req.LogoURL)
	}
	if req.StampURL != "" {
		v.CloudinaryURL("stamp_url", req.StampURL)
	}
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Motto != "" {
		updates["motto"] = req.Motto
	}
	if req.Address != "" {
		updates["address"] = req.Address
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.LogoURL != "" {
		updates["logo_url"] = req.LogoURL
	}
	if req.StampURL != "" {
		updates["stamp_url"] = req.StampURL
	}
	if req.SignaturePrincipalURL != "" {
		updates["signature_principal_url"] = req.SignaturePrincipalURL
	}
	if req.SignatureHeadteacherURL != "" {
		updates["signature_headteacher_url"] = req.SignatureHeadteacherURL
	}
	if req.PrimaryColor != "" {
		updates["primary_color"] = req.PrimaryColor
	}
	if req.Prefix != "" {
		updates["prefix"] = req.Prefix
	}
	if req.WatermarkEnabled != nil {
		updates["watermark_enabled"] = *req.WatermarkEnabled
	}
	if req.BursarName != "" {
		updates["bursar_name"] = req.BursarName
	}
	if req.AdmissionDocumentsList != "" {
		updates["admission_documents_list"] = req.AdmissionDocumentsList
	}
	if req.SchoolDirections != "" {
		updates["school_directions"] = req.SchoolDirections
	}
	if req.MaxVideoUploadMB != nil {
		updates["max_video_upload_mb"] = *req.MaxVideoUploadMB
	}
	if req.EstablishedYear != nil && *req.EstablishedYear > 1900 {
		updates["established_year"] = *req.EstablishedYear
	}

	if err := h.db.Model(&models.School{}).Where("id = ?", claims.SchoolID).Updates(updates).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update settings")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "Settings updated successfully", nil)
}

// GetGradingScales handles GET /api/settings/grading
func (h *SettingsHandler) GetGradingScales(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var scales []models.GradeScale
	h.db.Where("school_id = ?", claims.SchoolID).Order("sort_order ASC").Find(&scales)

	utils.RespondSuccess(w, http.StatusOK, "", scales)
}

// UpsertGradingScales handles PUT /api/settings/grading
func (h *SettingsHandler) UpsertGradingScales(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req struct {
		Scales []struct {
			Grade     string  `json:"grade"`
			MinScore  float64 `json:"min_score"`
			MaxScore  float64 `json:"max_score"`
			Remark    string  `json:"remark"`
			IsPassing bool    `json:"is_passing"`
			SortOrder int     `json:"sort_order"`
		} `json:"scales"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Delete existing scales and recreate (simple upsert strategy)
	h.db.Where("school_id = ?", claims.SchoolID).Delete(&models.GradeScale{})

	for _, s := range req.Scales {
		scale := models.GradeScale{
			SchoolID:  claims.SchoolID,
			Grade:     s.Grade,
			MinScore:  s.MinScore,
			MaxScore:  s.MaxScore,
			Remark:    s.Remark,
			IsPassing: s.IsPassing,
			SortOrder: s.SortOrder,
		}
		h.db.Create(&scale)
	}

	utils.RespondSuccess(w, http.StatusOK, "Grading scales updated", nil)
}
