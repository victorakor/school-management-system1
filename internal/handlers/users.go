package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"school-platform/internal/auth"
	"school-platform/internal/config"
	"school-platform/internal/jobs"
	"school-platform/internal/middleware"
	"school-platform/internal/models"
	"school-platform/internal/utils"
	"school-platform/internal/validators"
)

// UsersHandler handles user management and authentication endpoints.
type UsersHandler struct {
	db        *gorm.DB
	cfg       *config.Config
	jobClient *jobs.Client
}

// NewUsersHandler creates a new UsersHandler.
func NewUsersHandler(db *gorm.DB, cfg *config.Config, jobClient *jobs.Client) *UsersHandler {
	return &UsersHandler{db: db, cfg: cfg, jobClient: jobClient}
}

// ─── Auth Endpoints ────────────────────────────────────────────────────────────

// Login handles POST /api/auth/login
func (h *UsersHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("email", req.Email, "Email")
	v.Email("email", req.Email)
	v.Required("password", req.Password, "Password")
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	var user models.User
	if err := h.db.Where("email = ? AND is_archived = false", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.RespondError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, "Login failed")
		return
	}

	if !user.IsActive {
		utils.RespondError(w, http.StatusForbidden, "Your account has been deactivated — please contact the school")
		return
	}

	if !user.IsVerified && user.Role == models.RoleParent {
		utils.RespondError(w, http.StatusForbidden, "Please verify your email address before logging in")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	pair, err := auth.GenerateTokenPair(&user, h.cfg)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	auth.SetTokenCookies(w, pair, h.cfg)

	// Update last login
	now := time.Now()
	h.db.Model(&user).Update("last_login", now)

	utils.RespondSuccess(w, http.StatusOK, "Login successful", map[string]interface{}{
		"user": map[string]interface{}{
			"id":             user.ID,
			"full_name":      user.FullName,
			"email":          user.Email,
			"role":           user.Role,
			"division_scope": user.DivisionScope,
		},
	})
}

// Logout handles POST /api/auth/logout
func (h *UsersHandler) Logout(w http.ResponseWriter, r *http.Request) {
	auth.ClearTokenCookies(w)
	utils.RespondSuccess(w, http.StatusOK, "Logged out successfully", nil)
}

// RefreshToken handles POST /api/auth/refresh
func (h *UsersHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshStr, err := auth.GetRefreshTokenFromRequest(r)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "No refresh token found")
		return
	}

	claims, err := auth.ParseRefreshToken(refreshStr, h.cfg)
	if err != nil {
		auth.ClearTokenCookies(w)
		utils.RespondError(w, http.StatusUnauthorized, "Session expired — please log in again")
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ? AND is_archived = false", claims.UserID).Error; err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "User not found")
		return
	}

	pair, err := auth.GenerateTokenPair(&user, h.cfg)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to refresh session")
		return
	}

	auth.SetTokenCookies(w, pair, h.cfg)
	utils.RespondSuccess(w, http.StatusOK, "Token refreshed", nil)
}

// RegisterParent handles POST /api/auth/register (parent self-registration)
func (h *UsersHandler) RegisterParent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FullName        string `json:"full_name"`
		Email           string `json:"email"`
		Phone           string `json:"phone"`
		Password        string `json:"password"`
		PasswordConfirm string `json:"password_confirm"`
		SchoolID        string `json:"school_id"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("full_name", req.FullName, "Full name")
	v.MinLength("full_name", req.FullName, 2, "Full name")
	v.Required("email", req.Email, "Email")
	v.Email("email", req.Email)
	v.Required("phone", req.Phone, "Phone number")
	v.Phone("phone", req.Phone)
	v.Required("password", req.Password, "Password")
	v.Password("password", req.Password)
	v.PasswordMatch("password_confirm", req.Password, req.PasswordConfirm)
	v.Required("school_id", req.SchoolID, "School ID")
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	schoolID, err := uuid.Parse(req.SchoolID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid school ID")
		return
	}

	// Check email uniqueness
	var existing models.User
	if err := h.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		utils.RespondError(w, http.StatusConflict, "An account with this email already exists")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	// Generate OTP
	otp, err := utils.GenerateOTP()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Registration failed")
		return
	}
	otpExpiry := time.Now().Add(15 * time.Minute)

	user := &models.User{
		SchoolID:      schoolID,
		FullName:      req.FullName,
		Email:         req.Email,
		Phone:         req.Phone,
		PasswordHash:  string(hash),
		Role:          models.RoleParent,
		DivisionScope: models.DivisionAll,
		IsActive:      true,
		IsVerified:    false,
		OTPCode:       otp,
		OTPExpiresAt:  &otpExpiry,
	}

	if err := h.db.Create(user).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	// Enqueue OTP email job asynchronously
	if h.jobClient != nil {
		_ = h.jobClient.EnqueueEmailOTP(user.ID.String(), user.Email, user.FullName, otp)
	}

	utils.RespondSuccess(w, http.StatusCreated, "Registration successful — please check your email for a verification code", map[string]interface{}{
		"user_id": user.ID,
	})
}

// VerifyOTP handles POST /api/auth/verify-otp
func (h *UsersHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID  string `json:"user_id"`
		OTPCode string `json:"otp_code"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "User not found")
		return
	}

	if user.IsVerified {
		utils.RespondSuccess(w, http.StatusOK, "Email already verified", nil)
		return
	}

	if user.OTPCode != req.OTPCode {
		utils.RespondError(w, http.StatusUnprocessableEntity, "Invalid verification code")
		return
	}

	if user.OTPExpiresAt != nil && time.Now().After(*user.OTPExpiresAt) {
		utils.RespondError(w, http.StatusUnprocessableEntity, "Verification code has expired — please request a new one")
		return
	}

	h.db.Model(&user).Updates(map[string]interface{}{
		"is_verified":    true,
		"otp_code":       "",
		"otp_expires_at": nil,
	})

	utils.RespondSuccess(w, http.StatusOK, "Email verified successfully", nil)
}

// ResendOTP handles POST /api/auth/resend-otp
func (h *UsersHandler) ResendOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "User not found")
		return
	}

	if user.IsVerified {
		utils.RespondSuccess(w, http.StatusOK, "Email already verified", nil)
		return
	}

	otp, err := utils.GenerateOTP()
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to generate OTP")
		return
	}
	otpExpiry := time.Now().Add(15 * time.Minute)

	h.db.Model(&user).Updates(map[string]interface{}{
		"otp_code":       otp,
		"otp_expires_at": otpExpiry,
	})

	// Enqueue OTP email job asynchronously
	if h.jobClient != nil {
		_ = h.jobClient.EnqueueEmailOTP(user.ID.String(), user.Email, user.FullName, otp)
	}

	utils.RespondSuccess(w, http.StatusOK, "Verification code resent", nil)
}

// ─── User Management Endpoints ─────────────────────────────────────────────────

// GetMe handles GET /api/users/me
func (h *UsersHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", claims.UserID).Error; err != nil {
		utils.RespondError(w, http.StatusNotFound, "User not found")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "", map[string]interface{}{
		"id":             user.ID,
		"full_name":      user.FullName,
		"email":          user.Email,
		"phone":          user.Phone,
		"role":           user.Role,
		"division_scope": user.DivisionScope,
		"avatar_url":     user.AvatarURL,
		"last_login":     user.LastLogin,
	})
}

// ListUsers handles GET /api/users (admin only)
func (h *UsersHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var users []models.User
	query := h.db.Where("school_id = ? AND is_archived = false", claims.SchoolID)

	// Filter by role if provided
	if role := r.URL.Query().Get("role"); role != "" {
		query = query.Where("role = ?", role)
	}

	// Filter by division scope
	if div := r.URL.Query().Get("division"); div != "" {
		query = query.Where("division_scope = ? OR division_scope = 'ALL'", div)
	}

	query.Order("full_name ASC").Find(&users)

	// Strip sensitive fields
	type userResponse struct {
		ID            uuid.UUID            `json:"id"`
		FullName      string               `json:"full_name"`
		Email         string               `json:"email"`
		Phone         string               `json:"phone"`
		Role          models.Role          `json:"role"`
		DivisionScope models.DivisionScope `json:"division_scope"`
		IsActive      bool                 `json:"is_active"`
		AvatarURL     string               `json:"avatar_url"`
		LastLogin     *time.Time           `json:"last_login"`
	}

	resp := make([]userResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, userResponse{
			ID:            u.ID,
			FullName:      u.FullName,
			Email:         u.Email,
			Phone:         u.Phone,
			Role:          u.Role,
			DivisionScope: u.DivisionScope,
			IsActive:      u.IsActive,
			AvatarURL:     u.AvatarURL,
			LastLogin:     u.LastLogin,
		})
	}

	utils.RespondSuccess(w, http.StatusOK, "", resp)
}

// CreateUser handles POST /api/users (admin creates staff accounts)
func (h *UsersHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	var req struct {
		FullName      string               `json:"full_name"`
		Email         string               `json:"email"`
		Phone         string               `json:"phone"`
		Password      string               `json:"password"`
		Role          models.Role          `json:"role"`
		DivisionScope models.DivisionScope `json:"division_scope"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validators.New()
	v.Required("full_name", req.FullName, "Full name")
	v.Required("email", req.Email, "Email")
	v.Email("email", req.Email)
	v.Required("password", req.Password, "Password")
	v.Password("password", req.Password)
	v.Required("role", string(req.Role), "Role")
	if v.HasErrors() {
		utils.RespondJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"errors": v.Errors()})
		return
	}

	// Check email uniqueness
	var existing models.User
	if err := h.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		utils.RespondError(w, http.StatusConflict, "An account with this email already exists")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	divScope := req.DivisionScope
	if divScope == "" {
		divScope = models.DivisionAll
	}

	user := &models.User{
		SchoolID:      claims.SchoolID,
		FullName:      req.FullName,
		Email:         req.Email,
		Phone:         req.Phone,
		PasswordHash:  string(hash),
		Role:          req.Role,
		DivisionScope: divScope,
		IsActive:      true,
		IsVerified:    true, // Admin-created accounts are pre-verified
	}

	if err := h.db.Create(user).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	utils.RespondSuccess(w, http.StatusCreated, "User created successfully", map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
	})
}

// UpdateUser handles PUT /api/users/:id
func (h *UsersHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req struct {
		FullName      string               `json:"full_name"`
		Phone         string               `json:"phone"`
		Role          models.Role          `json:"role"`
		DivisionScope models.DivisionScope `json:"division_scope"`
		IsActive      *bool                `json:"is_active"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if req.FullName != "" {
		updates["full_name"] = req.FullName
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}
	if req.DivisionScope != "" {
		updates["division_scope"] = req.DivisionScope
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := h.db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "User updated successfully", nil)
}

// ArchiveUser handles DELETE /api/users/:id (archives, never deletes)
func (h *UsersHandler) ArchiveUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := h.db.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{
			"is_archived": true,
			"is_active":   false,
		}).Error; err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to archive user")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "User archived successfully", nil)
}
