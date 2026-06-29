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

// ─── Finance Reports (Owner / Bursar) ─────────────────────────────────────────

// GetFinanceSummary handles GET /api/finance/summary
// Returns school-wide income, outstanding debts, and payment counts.
// Owner sees all divisions; Bursar sees only their DivisionScope.
func (h *FinanceHandler) GetFinanceSummary(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	type summaryResult struct {
		TotalCollected   float64 `json:"total_collected"`
		TotalOutstanding float64 `json:"total_outstanding"`
		DebtorCount      int64   `json:"debtor_count"`
		PaymentCount     int64   `json:"payment_count"`
	}

	query := h.db.Table("fee_payments fp").
		Joins("JOIN students s ON s.id = fp.student_id").
		Where("s.school_id = ? AND fp.is_archived = false", claims.SchoolID)

	// Scope Bursar to their division
	if claims.Role == models.RoleBursar && claims.DivisionScope != models.DivisionAll {
		query = query.
			Joins("JOIN divisions d ON d.id = s.division_id").
			Where("d.name = ?", claims.DivisionScope)
	}

	var result summaryResult
	query.Select(`
		COALESCE(SUM(fp.amount_paid), 0) AS total_collected,
		COALESCE(SUM(CASE WHEN fp.balance_after > 0 THEN fp.balance_after ELSE 0 END), 0) AS total_outstanding,
		COUNT(CASE WHEN fp.balance_after > 0 THEN 1 END) AS debtor_count,
		COUNT(*) AS payment_count`).
		Scan(&result)

	utils.RespondSuccess(w, http.StatusOK, "", result)
}

// GetIncomeReport handles GET /api/finance/reports/income
// Returns payments grouped by date range. Query params: from, to (YYYY-MM-DD).
func (h *FinanceHandler) GetIncomeReport(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	query := h.db.Table("fee_payments fp").
		Joins("JOIN students s ON s.id = fp.student_id").
		Where("s.school_id = ? AND fp.is_archived = false", claims.SchoolID)

	if from := r.URL.Query().Get("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			query = query.Where("fp.payment_date >= ?", t)
		}
	}
	if to := r.URL.Query().Get("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			query = query.Where("fp.payment_date < ?", t.Add(24*time.Hour))
		}
	}
	if div := r.URL.Query().Get("division"); div != "" {
		query = query.
			Joins("JOIN divisions d ON d.id = s.division_id").
			Where("d.name = ?", div)
	}

	type paymentRow struct {
		PaymentDate   time.Time `json:"payment_date"`
		StudentName   string    `json:"student_name"`
		CategoryName  string    `json:"category_name"`
		AmountPaid    float64   `json:"amount_paid"`
		AmountOwed    float64   `json:"amount_owed"`
		BalanceAfter  float64   `json:"balance_after"`
		ReceiptRef    string    `json:"receipt_ref"`
	}

	var payments []paymentRow
	query.Select(`
		fp.payment_date, s.full_name AS student_name,
		fp.category_name, fp.amount_paid, fp.amount_owed,
		fp.balance_after, fp.receipt_ref`).
		Order("fp.payment_date DESC").
		Limit(500).
		Scan(&payments)

	if payments == nil {
		payments = []paymentRow{}
	}

	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"payments": payments,
		"count":    len(payments),
	})
}

// GetDebtReport handles GET /api/finance/reports/debts
// Returns students with outstanding balances.
func (h *FinanceHandler) GetDebtReport(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	type debtRow struct {
		StudentID    string  `json:"student_id"`
		StudentName  string  `json:"student_name"`
		AdmissionID  string  `json:"admission_id"`
		CategoryName string  `json:"category_name"`
		AmountOwed   float64 `json:"amount_owed"`
		AmountPaid   float64 `json:"amount_paid"`
		BalanceAfter float64 `json:"balance_after"`
	}

	query := h.db.Table("fee_payments fp").
		Joins("JOIN students s ON s.id = fp.student_id").
		Where("s.school_id = ? AND fp.is_archived = false AND fp.balance_after > 0", claims.SchoolID)

	if div := r.URL.Query().Get("division"); div != "" {
		query = query.
			Joins("JOIN divisions d ON d.id = s.division_id").
			Where("d.name = ?", div)
	}

	var debts []debtRow
	query.Select(`
		s.id AS student_id, s.full_name AS student_name, s.admission_id,
		fp.category_name, fp.amount_owed, fp.amount_paid, fp.balance_after`).
		Order("fp.balance_after DESC").
		Scan(&debts)

	if debts == nil {
		debts = []debtRow{}
	}

	var totalOutstanding float64
	for _, d := range debts {
		totalOutstanding += d.BalanceAfter
	}

	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"debts":             debts,
		"total_outstanding": totalOutstanding,
		"debtor_count":      len(debts),
	})
}
