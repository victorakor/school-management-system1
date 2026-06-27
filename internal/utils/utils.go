package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode"
)

// ─── Reference Number Generation ──────────────────────────────────────────────

// GenerateDocRef generates a unique alphanumeric document reference number.
// Format: [PREFIX]-[YEAR]-[8 random hex chars]
// Example: DOC-2025-A3F9B21C
func GenerateDocRef(prefix string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%d-%s", strings.ToUpper(prefix), time.Now().Year(), strings.ToUpper(hex.EncodeToString(b)))
}

// GenerateAdmissionID generates a permanent admission ID.
// Format: [SCHOOL_PREFIX]/[DIVISION_CODE]/[YEAR]/[SEQUENCE]
// Example: GRA/NUR/2025/001
func GenerateAdmissionID(schoolPrefix, divisionCode string, sequence int) string {
	return fmt.Sprintf("%s/%s/%d/%03d",
		strings.ToUpper(schoolPrefix),
		strings.ToUpper(divisionCode),
		time.Now().Year(),
		sequence,
	)
}

// GenerateOTP generates a 6-digit numeric OTP.
func GenerateOTP() (string, error) {
	max := big.NewInt(999999)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// GenerateReceiptRef generates a unique receipt reference.
// Format: RCP-[YEAR][MONTH]-[6 random hex chars]
func GenerateReceiptRef() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	now := time.Now()
	return fmt.Sprintf("RCP-%d%02d-%s", now.Year(), now.Month(), strings.ToUpper(hex.EncodeToString(b)))
}

// GenerateSecureToken generates a cryptographically secure random hex token of n bytes.
func GenerateSecureToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ─── Date Helpers ──────────────────────────────────────────────────────────────

// FormatDate formats a time.Time to "2 January 2006" (human-readable).
func FormatDate(t time.Time) string {
	return t.Format("2 January 2006")
}

// FormatDateTime formats a time.Time to "2 January 2006, 3:04 PM".
func FormatDateTime(t time.Time) string {
	return t.Format("2 January 2006, 3:04 PM")
}

// FormatTime formats a time.Time to "3:04 PM".
func FormatTime(t time.Time) string {
	return t.Format("3:04 PM")
}

// IsWeekday returns true if the given time is Monday–Friday.
func IsWeekday(t time.Time) bool {
	wd := t.Weekday()
	return wd != time.Saturday && wd != time.Sunday
}

// NextWeekday returns the next weekday (Mon–Fri) on or after the given time.
func NextWeekday(t time.Time) time.Time {
	for !IsWeekday(t) {
		t = t.AddDate(0, 0, 1)
	}
	return t
}

// ─── String Helpers ────────────────────────────────────────────────────────────

// TitleCase converts a string to title case (first letter of each word capitalised).
func TitleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			runes := []rune(w)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

// Truncate truncates a string to maxLen characters, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Initials returns the first letter of each word in a name, uppercased.
// e.g. "John Doe" → "J.D."
func Initials(name string) string {
	words := strings.Fields(name)
	var sb strings.Builder
	for _, w := range words {
		if len(w) > 0 {
			sb.WriteRune(unicode.ToUpper([]rune(w)[0]))
			sb.WriteRune('.')
		}
	}
	return sb.String()
}

// MaskName returns first name + initials of remaining names for privacy.
// e.g. "Emeka Chukwu Obi" → "Emeka C.O."
func MaskName(fullName string) string {
	words := strings.Fields(fullName)
	if len(words) == 0 {
		return ""
	}
	if len(words) == 1 {
		return words[0]
	}
	first := words[0]
	var sb strings.Builder
	sb.WriteString(first)
	sb.WriteString(" ")
	for _, w := range words[1:] {
		if len(w) > 0 {
			sb.WriteRune(unicode.ToUpper([]rune(w)[0]))
			sb.WriteRune('.')
		}
	}
	return sb.String()
}

// ─── Number Helpers ────────────────────────────────────────────────────────────

// OrdinalSuffix returns the ordinal suffix for a number (1st, 2nd, 3rd, 4th...).
func OrdinalSuffix(n int) string {
	switch {
	case n%100 >= 11 && n%100 <= 13:
		return fmt.Sprintf("%dth", n)
	case n%10 == 1:
		return fmt.Sprintf("%dst", n)
	case n%10 == 2:
		return fmt.Sprintf("%dnd", n)
	case n%10 == 3:
		return fmt.Sprintf("%drd", n)
	default:
		return fmt.Sprintf("%dth", n)
	}
}

// RoundFloat rounds a float64 to the specified number of decimal places.
func RoundFloat(val float64, precision int) float64 {
	if precision == 0 {
		return float64(int(val + 0.5))
	}
	pow := 1.0
	for i := 0; i < precision; i++ {
		pow *= 10
	}
	return float64(int(val*pow+0.5)) / pow
}

// ─── Appointment Slot Helpers ──────────────────────────────────────────────────

// AppointmentTimeSlots returns the available appointment time slots (9 AM – 2 PM).
var AppointmentTimeSlots = []string{
	"09:00 AM", "09:30 AM", "10:00 AM", "10:30 AM",
	"11:00 AM", "11:30 AM", "12:00 PM", "12:30 PM",
	"01:00 PM", "01:30 PM", "02:00 PM",
}

// DivisionCode returns the short code for a division name.
func DivisionCode(division string) string {
	switch strings.ToUpper(division) {
	case "NURSERY":
		return "NUR"
	case "PRIMARY":
		return "PRI"
	case "SECONDARY":
		return "SEC"
	default:
		return "UNK"
	}
}
