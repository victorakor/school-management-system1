package handlers

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"school-platform/internal/config"
	"school-platform/internal/middleware"
	"school-platform/internal/models"
	"school-platform/internal/utils"
)

// DocumentsHandler handles Cloudinary upload signing and document management.
type DocumentsHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewDocumentsHandler creates a new DocumentsHandler.
func NewDocumentsHandler(db *gorm.DB, cfg *config.Config) *DocumentsHandler {
	return &DocumentsHandler{db: db, cfg: cfg}
}

// SignUpload handles POST /api/upload/sign
// Returns a signed Cloudinary upload signature for direct browser-to-Cloudinary uploads.
func (h *DocumentsHandler) SignUpload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Folder    string `json:"folder"`
		AssetType string `json:"asset_type"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Folder == "" {
		req.Folder = "uploads"
	}
	if req.AssetType == "" {
		req.AssetType = "image"
	}

	timestamp := time.Now().Unix()
	params := map[string]string{
		"folder":    req.Folder,
		"timestamp": fmt.Sprintf("%d", timestamp),
	}

	sig := cloudinarySign(params, h.cfg.CloudinaryAPISecret)

	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"signature":  sig,
		"timestamp":  timestamp,
		"cloud_name": h.cfg.CloudinaryCloudName,
		"api_key":    h.cfg.CloudinaryAPIKey,
		"folder":     req.Folder,
	})
}

// cloudinarySign generates a Cloudinary API signature.
func cloudinarySign(params map[string]string, apiSecret string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	toSign := strings.Join(parts, "&") + apiSecret

	h := hmac.New(sha1.New, []byte(apiSecret))
	h.Write([]byte(toSign))
	return hex.EncodeToString(h.Sum(nil))
}

// ValidateCloudinaryURL validates that a URL is from Cloudinary.
func ValidateCloudinaryURL(url string) bool {
	return strings.HasPrefix(url, "https://res.cloudinary.com/")
}

// GetDocumentsByStudent handles GET /api/documents/student
func (h *DocumentsHandler) GetDocumentsByStudent(w http.ResponseWriter, r *http.Request) {
	studentIDStr := r.URL.Query().Get("student_id")
	if studentIDStr == "" {
		utils.RespondError(w, http.StatusBadRequest, "student_id is required")
		return
	}
	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	var results []models.Result
	h.db.Where("student_id = ? AND doc_ref != ''", studentID).Find(&results)

	var payments []models.FeePayment
	h.db.Where("student_id = ? AND doc_ref != ''", studentID).Find(&payments)

	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"report_cards": results,
		"fee_receipts": payments,
	})
}

// UploadAdmissionLetter handles POST /api/admissions/applications/letter
func (h *DocumentsHandler) UploadAdmissionLetter(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	appIDStr := r.URL.Query().Get("application_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	var req struct {
		FileURL string `json:"file_url"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if !ValidateCloudinaryURL(req.FileURL) {
		utils.RespondError(w, http.StatusBadRequest, "Invalid file URL — must be a Cloudinary URL")
		return
	}

	docRef := utils.GenerateDocRef("ADM")
	now := time.Now()
	letter := &models.AdmissionLetter{
		ApplicationID: appID,
		UploadedBy:    claims.UserID,
		FileURL:       req.FileURL,
		DocRef:        docRef,
		GeneratedAt:   now,
	}

	if err := h.db.Create(letter).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to save admission letter")
		return
	}

	h.db.Create(&models.DocumentVerification{
		DocRef:      docRef,
		DocType:     models.DocTypeAdmissionLetter,
		EntityID:    appID.String(),
		GeneratedAt: now,
	})

	utils.RespondSuccess(w, http.StatusCreated, "Admission letter uploaded", map[string]interface{}{
		"doc_ref":  docRef,
		"file_url": req.FileURL,
	})
}
