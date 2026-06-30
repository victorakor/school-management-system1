package handlers

import (
	"html/template"
	"net/http"
	"time"

	"school-platform/internal/auth"
	"school-platform/internal/config"
	"school-platform/internal/middleware"
	"school-platform/internal/models"

	"gorm.io/gorm"
)

// FeedPostData is the minimal feed post shape the homepage template needs.
type FeedPostData struct {
	Caption     string
	DivisionTag string
	MediaURLs   []string
}

// AnnouncementData is the minimal announcement shape the ticker needs.
type AnnouncementData struct {
	Title string
}

// PageData is injected into every server-rendered HTML page.
// It covers every {{.Field}} used across all public templates.
type PageData struct {
	// School identity
	SchoolName         string
	SchoolMotto        string
	SchoolLogoURL      string
	SchoolAddress      string
	SchoolPhone        string
	SchoolEmail        string
	SchoolPrimaryColor string
	SchoolID           string

	// Request
	CSRFToken       string
	Title           string
	MetaDescription string

	// Dates
	CurrentYear int

	// Admissions
	AdmissionWindowOpen      bool
	AdmissionWindowCloseDate string
	CurrentSessionName       string

	// Homepage extras
	TotalStudents    int
	TotalTeachers    int
	TotalStaff       int // all active staff roles combined (used on about page)
	YearsEstablished int
	VideoHeroURL     string
	AdmissionsOpen   bool // alias of AdmissionWindowOpen used in some templates

	// Feed preview (homepage only)
	FeedPosts []FeedPostData

	// Announcement ticker (homepage only)
	Announcements []AnnouncementData
}

// PagesHandler serves all server-rendered HTML pages.
type PagesHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewPagesHandler(db *gorm.DB, cfg *config.Config) *PagesHandler {
	return &PagesHandler{db: db, cfg: cfg}
}

// buildPageData loads school record + admission window + stats from DB.
// All DB errors are swallowed — sensible defaults always render.
func (h *PagesHandler) buildPageData(r *http.Request) PageData {
	data := PageData{
		// Hard-coded LEAPS defaults (overridden by DB below)
		SchoolName:         "Leadership Preparatory Academy – LEAPS",
		SchoolMotto:        "Building Tomorrow's World Now",
		SchoolAddress:      "Makurdi, Benue State, Nigeria",
		SchoolPrimaryColor: "#0F2557",
		CurrentYear:        time.Now().Year(),
		YearsEstablished:   5,
		CSRFToken:          middleware.GetCSRFToken(r),
	}

	if h.db == nil {
		return data
	}

	// ── School record ──────────────────────────────────────────────────────
	var school models.School
	if err := h.db.First(&school).Error; err == nil {
		data.SchoolID = school.ID.String()
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
		if school.EstablishedYear > 0 {
			data.YearsEstablished = time.Now().Year() - school.EstablishedYear
			if data.YearsEstablished < 1 {
				data.YearsEstablished = 1
			}
		}
	}

	// ── Active admission window ────────────────────────────────────────────
	var window models.AdmissionWindow
	if err := h.db.Preload("Session").Where("is_active = true").First(&window).Error; err == nil {
		data.AdmissionWindowOpen = true
		data.AdmissionsOpen = true
		data.AdmissionWindowCloseDate = window.CloseDate.Format("2 January 2006")
		if window.Session.Name != "" {
			data.CurrentSessionName = window.Session.Name
		}
	}

	// ── Active session name (even if window closed) ────────────────────────
	if data.CurrentSessionName == "" {
		var session models.AcademicSession
		if err := h.db.Where("is_active = true").First(&session).Error; err == nil {
			data.CurrentSessionName = session.Name
		}
	}

	// ── Student + teacher counts ───────────────────────────────────────────
	var studentCount int64
	h.db.Model(&models.Student{}).Where("is_active = true AND is_archived = false").Count(&studentCount)
	data.TotalStudents = int(studentCount)

	var teacherCount int64
	h.db.Model(&models.User{}).Where("role IN ? AND is_active = true AND is_archived = false",
		[]string{string(models.RoleTeacher), string(models.RoleClassTeacher)},
	).Count(&teacherCount)
	data.TotalTeachers = int(teacherCount)

	// All active staff (every role except students, pupils, parents) — used on the about page.
	var staffCount int64
	h.db.Model(&models.User{}).Where("role NOT IN ? AND is_active = true AND is_archived = false",
		[]string{
			string(models.RoleStudent),
			string(models.RolePupil),
			string(models.RoleParent),
		},
	).Count(&staffCount)
	data.TotalStaff = int(staffCount)

	// ── Latest 6 published feed posts for homepage preview ─────────────────
	var posts []models.ActivityPost
	if err := h.db.Where("is_published = true AND is_archived = false").
		Order("created_at desc").Limit(6).Find(&posts).Error; err == nil {
		for _, p := range posts {
			fp := FeedPostData{
				Caption:     p.Caption,
				DivisionTag: string(p.DivisionTag),
			}
			for _, m := range p.MediaURLs {
				fp.MediaURLs = append(fp.MediaURLs, m.URL)
			}
			data.FeedPosts = append(data.FeedPosts, fp)
		}
	}

	// ── Latest 5 published announcements for ticker ────────────────────────
	var announcements []models.Announcement
	if err := h.db.Where("is_published = true AND is_archived = false").
		Order("created_at desc").Limit(5).Find(&announcements).Error; err == nil {
		for _, a := range announcements {
			data.Announcements = append(data.Announcements, AnnouncementData{Title: a.Title})
		}
	}

	return data
}

// ServePage renders a standalone HTML template file with full PageData.
func (h *PagesHandler) ServePage(tmplPath, title, metaDesc string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := h.buildPageData(r)
		data.Title = title
		data.MetaDescription = metaDesc

		tmpl, err := template.ParseFiles(tmplPath)
		if err != nil {
			http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if execErr := tmpl.Execute(w, data); execErr != nil {
			// Headers already sent — can't change status; log via zerolog if available
			_ = execErr
		}
	}
}

// portalTemplateForRole maps a role to the correct role-specific portal template.
// Each template bakes in only the nav sections that role is permitted to see.
func portalTemplateForRole(role models.Role) (string, string) {
	switch role {
	case models.RoleOwner:
		return "web/templates/portal/portal-owner.html", "Owner Dashboard"
	case models.RolePrincipal, models.RoleVicePrincipal,
		models.RoleHeadTeacher, models.RoleAsstHeadTeacher:
		return "web/templates/portal/portal-admin.html", "Admin Portal"
	case models.RoleTeacher, models.RoleClassTeacher, models.RoleExamOfficer:
		return "web/templates/portal/portal-teacher.html", "Teacher Portal"
	case models.RoleAdmissionsOfficer, models.RoleBursar,
		models.RoleBlogManager, models.RoleICTAdmin:
		return "web/templates/portal/portal-staff.html", "Staff Portal"
	case models.RoleStudent, models.RolePupil:
		return "web/templates/portal/portal-student.html", "Student Portal"
	case models.RoleParent:
		return "web/templates/portal/portal-parent.html", "Parent Portal"
	default:
		// Fallback: most restricted template so unknown roles never see admin sections.
		return "web/templates/portal/portal-student.html", "Portal"
	}
}

// resolvePortalClaims figures out who is making this request.
//
// The /portal/* routes are server-rendered HTML pages and are intentionally
// NOT wrapped in middleware.Authenticate (an unauthenticated visit must still
// return the HTML shell so client-side JS can redirect to /login — Authenticate
// would instead short-circuit with a raw JSON 401). That means
// middleware.GetClaims(r) is always nil here. We parse the access token
// directly so the correct role-specific template (owner, admin, teacher, …)
// is served instead of silently falling back to the student template for
// every signed-in user, regardless of role.
func (h *PagesHandler) resolvePortalClaims(r *http.Request) *auth.Claims {
	// Prefer claims already in context, in case Authenticate is ever added
	// to this route group in the future.
	if claims := middleware.GetClaims(r); claims != nil {
		return claims
	}

	if h.cfg == nil {
		return nil
	}

	tokenStr, err := auth.GetAccessTokenFromRequest(r)
	if err != nil {
		return nil
	}

	claims, err := auth.ParseAccessToken(tokenStr, h.cfg)
	if err != nil {
		// Access token may be expired; the client will hit /api/users/me,
		// which DOES run through Authenticate and will transparently refresh
		// it. We don't attempt a refresh here since we can't reliably set
		// cookies once the body may already be streaming.
		return nil
	}

	return claims
}

// ServePortal reads the user's role from the JWT cookie and serves the matching
// role-specific portal template. Each template embeds only the nav sections
// that role is permitted to access, enforcing RBAC at the HTML delivery layer.
func (h *PagesHandler) ServePortal(w http.ResponseWriter, r *http.Request) {
	data := h.buildPageData(r)

	tmplPath := "web/templates/portal/portal-student.html"
	data.Title = "Portal"

	if claims := h.resolvePortalClaims(r); claims != nil {
		path, title := portalTemplateForRole(claims.Role)
		tmplPath = path
		data.Title = title
	}

	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "portal template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}
