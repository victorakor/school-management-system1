package handlers

import (
	"net/http"

	"gorm.io/gorm"

	"school-platform/internal/models"
	"school-platform/internal/utils"
)

// VerifyHandler handles the public document verification endpoint.
type VerifyHandler struct {
	db *gorm.DB
}

// NewVerifyHandler creates a new VerifyHandler.
func NewVerifyHandler(db *gorm.DB) *VerifyHandler {
	return &VerifyHandler{db: db}
}

// VerifyDocument handles GET /api/verify?ref=XXXXX
func (h *VerifyHandler) VerifyDocument(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		utils.RespondError(w, http.StatusBadRequest, "Document reference number is required")
		return
	}

	var doc models.DocumentVerification
	if err := h.db.Where("doc_ref = ?", ref).First(&doc).Error; err != nil {
		utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
			"verified":    false,
			"status":      "NOT_FOUND",
			"message":     "No document found with this reference number",
		})
		return
	}

	// Increment verified count
	h.db.Model(&doc).Update("verified_count", doc.VerifiedCount+1)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"verified":     true,
		"status":       "VERIFIED",
		"doc_type":     doc.DocType,
		"student_name": utils.MaskName(doc.StudentName), // privacy: first name + initials
		"issued_by":    doc.IssuedBy,
		"generated_at": utils.FormatDateTime(doc.GeneratedAt),
		"verified_count": doc.VerifiedCount + 1,
	})
}
