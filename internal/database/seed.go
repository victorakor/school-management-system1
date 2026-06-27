package database

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"school-platform/internal/models"
)

// Seed runs a one-time idempotent seed for the default school and its
// Owner user.  It is safe to call on every startup — rows that already
// exist are left untouched.
//
// The Owner's default password is "ChangeMe!2026".  Rotate it on first login.
func Seed(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("seed: nil db")
	}

	// ── Default school ────────────────────────────────────────────────────
	var school models.School
	schoolID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	err := db.First(&school, "id = ?", schoolID).Error
	if err == gorm.ErrRecordNotFound {
		school = models.School{
			Base:                models.Base{ID: schoolID},
			Name:                "Grace Academy",
			Motto:               "Knowledge with Discipline",
			Address:             "Plot 24, Independence Avenue, Abuja, Nigeria",
			Phone:               "+234-800-000-0001",
			Email:               "info@graceacademy.test",
			PrimaryColor:        "#0F2557",
			Prefix:              "GRA",
			WatermarkEnabled:    false,
			MaxVideoUploadMB:    100,
		}
		if err := db.Create(&school).Error; err != nil {
			return fmt.Errorf("seed: create school: %w", err)
		}
		logSeed("created default school: %s", school.Name)
	} else if err != nil {
		return fmt.Errorf("seed: lookup school: %w", err)
	}

	// ── Divisions ──────────────────────────────────────────────────────────
	divisions := []struct {
		id   uuid.UUID
		name models.DivisionScope
	}{
		{uuid.MustParse("00000000-0000-0000-0000-000000000021"), models.DivisionNursery},
		{uuid.MustParse("00000000-0000-0000-0000-000000000022"), models.DivisionPrimary},
		{uuid.MustParse("00000000-0000-0000-0000-000000000023"), models.DivisionSecondary},
	}
	for _, d := range divisions {
		var existing models.Division
		if err := db.First(&existing, "id = ?", d.id).Error; err == gorm.ErrRecordNotFound {
			div := models.Division{
				Base:     models.Base{ID: d.id},
				SchoolID: schoolID,
				Name:     d.name,
			}
			if err := db.Create(&div).Error; err != nil {
				return fmt.Errorf("seed: create division %s: %w", d.name, err)
			}
			logSeed("created division: %s", d.name)
		}
	}

	// ── Default active academic session ────────────────────────────────────
	var activeSession models.AcademicSession
	if err := db.First(&activeSession, "school_id = ? AND is_active = true", schoolID).Error; err == gorm.ErrRecordNotFound {
		now := time.Now().UTC()
		session := models.AcademicSession{
			Base:       models.Base{ID: uuid.MustParse("00000000-0000-0000-0000-000000000030")},
			SchoolID:   schoolID,
			Name:       fmt.Sprintf("%d/%d", now.Year(), now.Year()+1),
			StartDate:  time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:    time.Date(now.Year()+1, 12, 31, 0, 0, 0, 0, time.UTC),
			IsActive:   true,
		}
		if err := db.Create(&session).Error; err != nil {
			return fmt.Errorf("seed: create session: %w", err)
		}
		logSeed("created active session: %s", session.Name)
	}

	// ── Default Owner user ─────────────────────────────────────────────────
	var owner models.User
	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	err = db.First(&owner, "id = ?", ownerID).Error
	if err == gorm.ErrRecordNotFound {
		pwHash, hashErr := bcrypt.GenerateFromPassword([]byte("ChangeMe!2026"), bcrypt.DefaultCost)
		if hashErr != nil {
			return fmt.Errorf("seed: hash owner password: %w", hashErr)
		}
		owner = models.User{
			Base:          models.Base{ID: ownerID},
			SchoolID:      schoolID,
			FullName:      "Default Owner",
			Email:         "owner@graceacademy.test",
			Phone:         "+234-800-000-0001",
			PasswordHash:  string(pwHash),
			Role:          models.RoleOwner,
			DivisionScope: models.DivisionAll,
			IsActive:      true,
			IsVerified:    true,
		}
		if err := db.Create(&owner).Error; err != nil {
			return fmt.Errorf("seed: create owner: %w", err)
		}
		logSeed("created default Owner (email=owner@graceacademy.test, password=ChangeMe!2026 — rotate on first login)")
	}

	return nil
}

func logSeed(format string, args ...interface{}) {
	// Use a structured-style message; the database package already pulls in
	// zerolog via the parent file.  Lightweight fmt here keeps the seeder
	// self-contained.
	fmt.Printf("[seed] "+format+"\n", args...)
}