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
	"school-platform/internal/utils"
	"school-platform/internal/validators"
)

// FinanceHandler handles fee structure, payment recording, and reports.
type FinanceHandler struct {
	db        *gorm.DB
	cfg       *config.Config
	jobClient *jobs.Client
}

// NewFinanceHandler creates a new FinanceHandler.
func NewFinanceHandler(db *gorm.DB, cfg *config.Config, jobClient *jobs.Client) *FinanceHandler {
	return &FinanceHandler{db: db, cfg: cfg, jobClient: jobClient}
}

// ─── Fee Structure ─────────────────────────────────────────────────────────────

// GetFeeStructure handles GET /api/finance/structure
func (h *FinanceHandler) GetFeeStructure(w http.ResponseWriter, r *http.Request) {
	divisionID, _ := uuid.Parse(r.URL.Query().Get("division_id"))
	sessionID, _ := uuid.Parse(r.URL.Query().Get("session_id"))
	termID, _ := uuid.Parse(r.URL.Query().Get("term_id"))

	var structure models.FeeStructure
	if err := h.db.Where("division_id = ? AND session_id = ? AND term_id = ?",
		divisionID, sessionID, termID).First(&structure).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "Fee structure not found")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "", structure)
}

// UpsertFeeStructure handles POST /api/finance/structure
func (h *FinanceHandler) UpsertFeeStructure(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req struct {
		DivisionID string `json:"division_id"`
		SessionID  string `json:"session_id"`
		TermID     string `json:"term_id"`
		Categories []struct {
			Name          string             `json:"name"`
			AmountByClass map[string]float64 `json:"amount_by_class"`
		} `json:"categories"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("division_id", req.DivisionID, "Division")
	v.Required("session_id", req.SessionID, "Session")
	v.Required("term_id", req.TermID, "Term")
	if len(req.Categories) == 0 {
		v.Custom("categories", "At least one fee category is required")
	}
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	divisionID, _ := uuid.Parse(req.DivisionID)
	sessionID, _ := uuid.Parse(req.SessionID)
	termID, _ := uuid.Parse(req.TermID)

	cats := make(models.JSONSlice, len(req.Categories))
	for i, c := range req.Categories {
		amtMap := make(map[string]interface{})
		for k, v := range c.AmountByClass {
			amtMap[k] = v
		}
		cats[i] = map[string]interface{}{
			"name":            c.Name,
			"amount_by_class": amtMap,
		}
	}

	var structure models.FeeStructure
	h.db.Where("division_id = ? AND session_id = ? AND term_id = ?",
		divisionID, sessionID, termID).FirstOrInit(&structure)

	structure.DivisionID = divisionID
	structure.SessionID = sessionID
	structure.TermID = termID
	structure.BursarID = claims.UserID
	structure.Categories = cats

	if err := h.db.Save(&structure).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to save fee structure")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Fee structure saved", structure)
}

// ─── Payments ──────────────────────────────────────────────────────────────────

// GetStudentFees handles GET /api/finance/student/:studentId — returns fee breakdown + payment history
func (h *FinanceHandler) GetStudentFees(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(chi.URLParam(r, "studentId"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	var payments []models.FeePayment
	h.db.Preload("FeeStructure").Preload("Recorder").
		Where("student_id = ? AND is_archived = false", studentID).
		Order("payment_date DESC").Find(&payments)

	// Calculate totals
	var totalOwed, totalPaid float64
	for _, p := range payments {
		totalOwed += p.AmountOwed
		totalPaid += p.AmountPaid
	}

	status := "FULLY_PAID"
	if totalPaid == 0 {
		status = "OWING"
	} else if totalPaid < totalOwed {
		status = "PART_PAYMENT"
	}

	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"payments":    payments,
		"total_owed":  totalOwed,
		"total_paid":  totalPaid,
		"balance":     totalOwed - totalPaid,
		"status":      status,
	})
}

// RecordPayment handles POST /api/finance/payments
func (h *FinanceHandler) RecordPayment(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req struct {
		StudentID      string  `json:"student_id"`
		FeeStructureID string  `json:"fee_structure_id"`
		CategoryName   string  `json:"category_name"`
		AmountOwed     float64 `json:"amount_owed"`
		AmountPaid     float64 `json:"amount_paid"`
		PaymentDate    string  `json:"payment_date"`
		Notes          string  `json:"notes"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("student_id", req.StudentID, "Student")
	v.Required("fee_structure_id", req.FeeStructureID, "Fee structure")
	v.Required("category_name", req.CategoryName, "Fee category")
	if req.AmountPaid <= 0 {
		v.Custom("amount_paid", "Amount paid must be greater than zero")
	}
	if req.AmountOwed <= 0 {
		v.Custom("amount_owed", "Amount owed must be greater than zero")
	}
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	studentID, _ := uuid.Parse(req.StudentID)
	feeStructureID, _ := uuid.Parse(req.FeeStructureID)

	paymentDate := time.Now()
	if req.PaymentDate != "" {
		paymentDate, _ = time.Parse("2006-01-02", req.PaymentDate)
	}

	// Calculate previous payments for this category to determine balance
	var prevPaid float64
	h.db.Model(&models.FeePayment{}).
		Where("student_id = ? AND fee_structure_id = ? AND category_name = ? AND is_archived = false",
			studentID, feeStructureID, req.CategoryName).
		Select("COALESCE(SUM(amount_paid), 0)").Scan(&prevPaid)

	balance := req.AmountOwed - (prevPaid + req.AmountPaid)

	payment := &models.FeePayment{
		StudentID:      studentID,
		FeeStructureID: feeStructureID,
		CategoryName:   req.CategoryName,
		AmountOwed:     req.AmountOwed,
		AmountPaid:     req.AmountPaid,
		PaymentDate:    paymentDate,
		RecordedBy:     claims.UserID,
		ReceiptRef:     utils.GenerateReceiptRef(),
		BalanceAfter:   balance,
		DocRef:         utils.GenerateDocRef("RCP"),
		Notes:          req.Notes,
	}

	if err := h.db.Create(payment).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to record payment")
		return
	}

	// Enqueue fee receipt PDF generation job
	if h.jobClient != nil {
		_ = h.jobClient.EnqueueFeeReceiptPDF(payment.ID.String(), claims.SchoolID.String())
	}

	utils.RespondSuccess(w, http.StatusCreated, "Payment recorded successfully", map[string]interface{}{
		"payment_id":  payment.ID,
		"receipt_ref": payment.ReceiptRef,
		"balance":     balance,
	})
}

// ─── Discounts ─────────────────────────────────────────────────────────────────

// ApplyDiscount handles POST /api/finance/discounts
func (h *FinanceHandler) ApplyDiscount(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req struct {
		StudentID      string  `json:"student_id"`
		FeeStructureID string  `json:"fee_structure_id"`
		Percentage     float64 `json:"percentage"`
		Reason         string  `json:"reason"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("student_id", req.StudentID, "Student")
	v.Required("reason", req.Reason, "Reason")
	if req.Percentage <= 0 || req.Percentage > 100 {
		v.Custom("percentage", "Discount percentage must be between 1 and 100")
	}
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	studentID, _ := uuid.Parse(req.StudentID)
	feeStructureID, _ := uuid.Parse(req.FeeStructureID)

	discount := &models.FeeDiscount{
		StudentID:      studentID,
		FeeStructureID: feeStructureID,
		Percentage:     req.Percentage,
		Reason:         req.Reason,
		AppliedBy:      claims.UserID,
	}

	if err := h.db.Create(discount).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to apply discount")
		return
	}
	utils.RespondSuccess(w, http.StatusCreated, "Discount applied", discount)
}
