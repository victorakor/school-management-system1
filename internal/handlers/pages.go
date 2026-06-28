package handlers

import (
	"html/template"
	"net/http"

	"school-platform/internal/middleware"
	"school-platform/internal/models"

	"gorm.io/gorm"
)

// PageData is the template data struct injected into every server-rendered page.
type PageData struct {
	SchoolName        string
	SchoolMotto       string
	SchoolLogoURL     string
	SchoolAddress     string
	SchoolPhone       string
	SchoolEmail       string
	SchoolPrimaryColor string
	CSRFToken         string
	Title             string
	MetaDescription   string
	// AdmissionWindowOpen is true when a public admission window is currently open.
	AdmissionWindowOpen bool
}

// PagesHandler serves all server-rendered HTML pages with injected school data.
type PagesHandler struct {
	db *gorm.DB
}

func NewPagesHandler(db *gorm.DB) *PagesHandler {
	return &PagesHandler{db: db}
}

// loadSchool fetches the first (only) school record. Falls back to LEAPS defaults
// if the DB is unavailable or the record has no name yet.
func (h *PagesHandler) loadSchool() PageData {
	data := PageData{
		SchoolName:         "Leadership Preparatory Academy – LEAPS",
		SchoolMotto:        "Building Tomorrow's World Now",
		SchoolAddress:      "Makurdi, Benue State, Nigeria",
		SchoolPhone:        "",
		SchoolEmail:        "",
		SchoolPrimaryColor: "#0F2557",
	}

	if h.db != nil {
		var school models.School
		if err := h.db.First(&school).Error; err == nil {
			if school.Name != "" {
				data.SchoolName = school.Name
			}
			if school.Motto != "" {
				data.SchoolMotto = school.Motto
			}
			if school.Address != "" {
				data.SchoolAddress = school.Address
			}
			if school.Phone != "" {
				data.SchoolPhone = school.Phone
			}
			if school.Email != "" {
				data.SchoolEmail = school.Email
			}
			if school.LogoURL != "" {
				data.SchoolLogoURL = school.LogoURL
			}
			if school.PrimaryColor != "" {
				data.SchoolPrimaryColor = school.PrimaryColor
			}
		}
	}
	return data
}

// ServePage renders a standalone HTML template file (not using the layout
// partial system — each public/auth page is self-contained) with PageData.
func (h *PagesHandler) ServePage(tmplPath, title, metaDesc string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := h.loadSchool()
		data.Title = title
		data.MetaDescription = metaDesc
		data.CSRFToken = middleware.GetCSRFToken(r)

		// Check admission window for pages that show it
		if h.db != nil {
			var window models.AdmissionWindow
			if err := h.db.Where("is_active = true").First(&window).Error; err == nil {
				data.AdmissionWindowOpen = true
			}
		}

		abs := tmplPath
		tmpl, err := template.ParseFiles(abs)
		if err != nil {
			http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// The template file name becomes the entry point for Execute
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if execErr := tmpl.Execute(w, data); execErr != nil {
			// Headers already sent — log only
			_ = execErr
		}
	}
}

// ServePortal renders portal.html with school data + CSRF token.
func (h *PagesHandler) ServePortal(w http.ResponseWriter, r *http.Request) {
	data := h.loadSchool()
	data.CSRFToken = middleware.GetCSRFToken(r)
	data.Title = "Portal"

	tmpl, err := template.ParseFiles("web/templates/portal.html")
	if err != nil {
		http.Error(w, "portal template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}

