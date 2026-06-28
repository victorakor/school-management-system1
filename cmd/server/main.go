package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"school-platform/internal/config"
	"school-platform/internal/database"
	"school-platform/internal/handlers"
	"school-platform/internal/jobs"
	"school-platform/internal/middleware"
	"school-platform/internal/models"
	"school-platform/internal/pdf"
	"school-platform/internal/permissions"
	"school-platform/internal/services"
)

func main() {
	// ─── Logger ───────────────────────────────────────────────────────────────
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// ─── Config ───────────────────────────────────────────────────────────────
	cfg := config.Load()
	log.Info().Str("env", cfg.Environment).Msg("starting school platform server")

	// ─── Sentry ───────────────────────────────────────────────────────────────
	if cfg.SentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Environment:      cfg.Environment,
			TracesSampleRate: 0.1,
			AttachStacktrace: true,
		}); err != nil {
			log.Warn().Err(err).Msg("sentry initialisation failed — error tracking disabled")
		} else {
			log.Info().Msg("sentry initialised")
			defer sentry.Flush(2 * time.Second)
		}
	}

	// ─── Database ─────────────────────────────────────────────────────────────
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	if err := database.RunMigrations(db); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	if err := database.Seed(db); err != nil {
		log.Fatal().Err(err).Msg("failed to seed database")
	}

	// Populate demo data for all roles so every portal feature can be tested.
	if err := database.DemoSeed(db); err != nil {
		log.Warn().Err(err).Msg("demo seed warning (non-fatal)")
	}

	// ─── Services ─────────────────────────────────────────────────────────────
	notifService := services.NewNotificationService(db)
	admissionService := services.NewAdmissionService(db, notifService)
	resultService := services.NewResultService(db, notifService)
	emailService := services.NewEmailService(cfg.ResendAPIKey, "noreply@"+extractHost(cfg.AppBaseURL))
	pdfGenerator := pdf.NewGenerator(cfg)
	jobClient := jobs.NewClient(cfg.RedisURL)

	// ─── Handlers ─────────────────────────────────────────────────────────────
	usersHandler := handlers.NewUsersHandler(db, cfg, jobClient)
	settingsHandler := handlers.NewSettingsHandler(db, cfg)
	academicHandler := handlers.NewAcademicHandler(db, cfg)
	verifyHandler := handlers.NewVerifyHandler(db)
	notifHandler := handlers.NewNotificationsHandler(db, notifService)
	admissionsHandler := handlers.NewAdmissionsHandler(db, cfg, admissionService, jobClient)
	scoresHandler := handlers.NewScoresHandler(db, cfg)
	resultsHandler := handlers.NewResultsHandler(db, cfg, resultService, jobClient)
	quizzesHandler := handlers.NewQuizzesHandler(db, cfg, jobClient)
	financeHandler := handlers.NewFinanceHandler(db, cfg, jobClient)
	feedHandler := handlers.NewFeedHandler(db, cfg)
	timetableHandler := handlers.NewTimetableHandler(db, cfg)
	attendanceHandler := handlers.NewAttendanceHandler(db, cfg)
	studentsHandler := handlers.NewStudentsHandler(db, cfg)
	announcementsHandler := handlers.NewAnnouncementsHandler(db, cfg)
	documentsHandler := handlers.NewDocumentsHandler(db, cfg)
	pagesHandler := handlers.NewPagesHandler(db)

	// ─── Router ───────────────────────────────────────────────────────────────
	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.RequestLogger)
	r.Use(middleware.SecurityHeaders)

	// ─── Health Check ─────────────────────────────────────────────────────────
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := database.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","error":"%s"}`, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","env":"%s"}`, cfg.Environment)
	})

	// ─── Static Assets ────────────────────────────────────────────────────────
	r.Handle("/static/*", middleware.StaticCache(
		http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))),
	))

	// ─── API Routes ───────────────────────────────────────────────────────────
	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.NoCache)

		// Public auth routes — tight rate limit to deter credential stuffing.
		authLimiter := middleware.RateLimit(10, time.Minute)
		r.Route("/auth", func(r chi.Router) {
			r.Use(authLimiter)
			r.With(middleware.Audit(db, "login", "user")).Post("/login", usersHandler.Login)
			r.Post("/logout", usersHandler.Logout)
			r.Post("/refresh", usersHandler.RefreshToken)
			r.With(middleware.Audit(db, "register", "user")).Post("/register", usersHandler.RegisterParent)
			r.Post("/verify-otp", usersHandler.VerifyOTP)
			r.Post("/resend-otp", usersHandler.ResendOTP)
		})

		// Public endpoints — moderate rate limit
		publicLimiter := middleware.RateLimit(60, time.Minute)
		r.With(publicLimiter).Get("/verify", verifyHandler.VerifyDocument)
		r.With(publicLimiter).Get("/feed", feedHandler.ListPosts)

		// ─── Protected Routes ──────────────────────────────────────────────────
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(cfg))

			// Current user
			r.Get("/users/me", usersHandler.GetMe)

			// User management
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermManageUsers))
				r.Get("/users", usersHandler.ListUsers)
				r.Post("/users", usersHandler.CreateUser)
				r.Put("/users/{id}", usersHandler.UpdateUser)
				r.Delete("/users/{id}", usersHandler.ArchiveUser)
			})

			// School settings
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermManageSchoolSettings))
				r.Get("/settings/school", settingsHandler.GetSchoolSettings)
				r.Put("/settings/school", settingsHandler.UpdateSchoolSettings)
				r.Get("/settings/grading", settingsHandler.GetGradingScales)
				r.Put("/settings/grading", settingsHandler.UpsertGradingScales)
			})

			// Sessions
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermManageSessions))
				r.Get("/sessions", academicHandler.ListSessions)
				r.Post("/sessions", academicHandler.CreateSession)
				r.Put("/sessions/{id}/activate", academicHandler.SetActiveSession)
			})

			// Terms
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermManageTerms))
				r.Get("/sessions/{sessionId}/terms", academicHandler.ListTerms)
				r.Post("/sessions/{sessionId}/terms", academicHandler.CreateTerm)
			})

			// Classes
			r.Get("/classes", academicHandler.ListClasses)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermManageClasses))
				r.Post("/classes", academicHandler.CreateClass)
			})

			// Subjects
			r.Get("/subjects", academicHandler.ListSubjects)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermManageSubjects))
				r.Post("/subjects", academicHandler.CreateSubject)
			})

			// Teacher assignments
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermAssignTeachers))
				r.Post("/assignments", academicHandler.AssignTeacher)
			})

			// Notifications
			r.Get("/notifications", notifHandler.ListNotifications)
			r.Get("/notifications/unread-count", notifHandler.GetUnreadCount)
			r.Put("/notifications/{id}/read", notifHandler.MarkRead)
			r.Put("/notifications/read-all", notifHandler.MarkAllRead)

			// Admissions
			r.Get("/admissions/window", admissionsHandler.GetAdmissionWindow)
			r.Get("/admissions/my-applications", admissionsHandler.GetParentApplications)
			r.Post("/admissions/applications", admissionsHandler.SubmitApplication)
			r.Post("/admissions/applications/{id}/reschedule", admissionsHandler.RescheduleAppointment)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermManageAdmissions))
				r.Post("/admissions/window", admissionsHandler.CreateAdmissionWindow)
				r.Get("/admissions/applications", admissionsHandler.ListApplications)
				r.Get("/admissions/applications/{id}", admissionsHandler.GetApplication)
				r.Put("/admissions/applications/{id}/status", admissionsHandler.UpdateApplicationStatus)
			})

			// Students
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermManageStudents))
				r.Get("/students", studentsHandler.ListStudents)
				r.Get("/students/{id}", studentsHandler.GetStudent)
				r.Put("/students/{id}", studentsHandler.UpdateStudent)
				r.Delete("/students/{id}", studentsHandler.ArchiveStudent)
				r.Post("/students/promote", studentsHandler.PromoteStudents)
			})

			// Scores
			r.Get("/scores/structure", scoresHandler.GetScoreStructure)
			r.Get("/scores/sheet", scoresHandler.GetScoreSheet)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermEnterScores))
				r.Post("/scores/structure", scoresHandler.UpsertScoreStructure)
				r.Post("/scores/save", scoresHandler.SaveScores)
				r.Post("/scores/{id}/unlock-request", scoresHandler.RequestUnlock)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermApproveScores))
				r.With(middleware.Audit(db, "approve_score", "score_entry")).Put("/scores/{id}/approve", scoresHandler.ApproveScores)
				r.With(middleware.Audit(db, "reject_score", "score_entry")).Put("/scores/{id}/reject", scoresHandler.RejectScores)
			})

			// Results
			r.Get("/results", resultsHandler.ListResults)
			r.Get("/results/{id}", resultsHandler.GetResult)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermManageResults))
				r.Post("/results/calculate", resultsHandler.CalculateResults)
				r.Put("/results/{id}/remarks", resultsHandler.UpdateRemarks)
				r.With(middleware.Audit(db, "publish_results", "result")).Post("/results/publish", resultsHandler.PublishResults)
			})

			// Quizzes
			r.Get("/quizzes", quizzesHandler.ListQuizzes)
			r.Get("/quizzes/questions", quizzesHandler.ListQuestions)
			r.Post("/quizzes/{id}/start", quizzesHandler.StartAttempt)
			r.Post("/quizzes/attempts/{id}/save", quizzesHandler.SaveProgress)
			r.Post("/quizzes/attempts/{id}/violation", quizzesHandler.LogViolation)
			r.Post("/quizzes/attempts/{id}/submit", quizzesHandler.SubmitAttempt)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermManageQuizzes))
				r.Post("/quizzes", quizzesHandler.CreateQuiz)
				r.Put("/quizzes/questions/{id}/approve", quizzesHandler.ApproveQuestion)
				r.Put("/quizzes/questions/{id}/flag", quizzesHandler.FlagQuestion)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermSubmitQuizQuestions))
				r.Post("/quizzes/questions", quizzesHandler.SubmitQuestion)
			})

			// Finance
			r.Get("/finance/structure", financeHandler.GetFeeStructure)
			r.Get("/finance/student/{studentId}", financeHandler.GetStudentFees)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermManageFinances))
				r.Post("/finance/structure", financeHandler.UpsertFeeStructure)
				r.With(middleware.Audit(db, "record_payment", "fee_payment")).Post("/finance/payments", financeHandler.RecordPayment)
				r.Post("/finance/discounts", financeHandler.ApplyDiscount)
			})

			// Feed (admin)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermPostActivityFeed))
				r.Post("/feed", feedHandler.CreatePost)
				r.Put("/feed/{id}", feedHandler.UpdatePost)
				r.Delete("/feed/{id}", feedHandler.ArchivePost)
			})

			// Timetable
			r.Get("/timetables", timetableHandler.ListTimetables)
			r.Get("/timetables/{id}", timetableHandler.GetTimetable)
			r.Get("/timetables/history", timetableHandler.GetTimetableHistory)
			r.Get("/timetables/check-conflict", timetableHandler.CheckTeacherConflict)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermManageTimetable))
				r.Post("/timetables/builder", timetableHandler.CreateBuilderTimetable)
				r.Post("/timetables/upload", timetableHandler.UploadTimetable)
			})

			// Attendance
			r.Get("/attendance", attendanceHandler.GetAttendance)
			r.Get("/attendance/summary", attendanceHandler.GetStudentAttendanceSummary)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermManageAttendance))
				r.Post("/attendance", attendanceHandler.MarkAttendance)
			})

			// Announcements
			r.Get("/announcements", announcementsHandler.ListAnnouncements)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(permissions.PermPublishBlogPosts))
				r.Post("/announcements", announcementsHandler.CreateAnnouncement)
				r.Put("/announcements/{id}", announcementsHandler.UpdateAnnouncement)
				r.Delete("/announcements/{id}", announcementsHandler.ArchiveAnnouncement)
			})

			// Documents / Upload signing
			r.Post("/upload/sign", documentsHandler.SignUpload)
			r.Get("/documents/student", documentsHandler.GetDocumentsByStudent)
			r.Post("/admissions/applications/letter", documentsHandler.UploadAdmissionLetter)

			// Dashboard stats
			dashboardHandler := handlers.NewDashboardHandler(db)
			r.Get("/dashboard/stats", dashboardHandler.Stats)
		})
	})

	// ─── Server-Rendered Pages ────────────────────────────────────────────────
	// All pages use middleware.CSRFProtect so the CSRF token is in context.
	r.Group(func(r chi.Router) {
		r.Use(middleware.CSRFProtect(cfg))
		r.Get("/", pagesHandler.ServePage("web/templates/public/index.html", "", "Leadership Preparatory Academy – LEAPS. Nursery, Primary and Secondary education in Makurdi, Benue State."))
		r.Get("/about", pagesHandler.ServePage("web/templates/public/about.html", "About Us", "Learn about Leadership Preparatory Academy – LEAPS, Makurdi, Benue State."))
		r.Get("/admissions", pagesHandler.ServePage("web/templates/public/admissions.html", "Admissions", "Apply to Leadership Preparatory Academy – LEAPS."))
		r.Get("/feed", pagesHandler.ServePage("web/templates/public/feed.html", "Activities", "School activities and news from LEAPS."))
		r.Get("/verify", pagesHandler.ServePage("web/templates/public/verify.html", "Verify Document", "Verify an official document issued by LEAPS."))
		r.Get("/divisions/nursery", pagesHandler.ServePage("web/templates/public/divisions/nursery.html", "Nursery Division", "Nursery division at Leadership Preparatory Academy."))
		r.Get("/divisions/primary", pagesHandler.ServePage("web/templates/public/divisions/primary.html", "Primary Division", "Primary division at Leadership Preparatory Academy."))
		r.Get("/divisions/secondary", pagesHandler.ServePage("web/templates/public/divisions/secondary.html", "Secondary Division", "Secondary division at Leadership Preparatory Academy."))
		r.Get("/login", pagesHandler.ServePage("web/templates/auth/login.html", "Sign In", "Sign in to the LEAPS portal."))
		r.Get("/register", pagesHandler.ServePage("web/templates/auth/register.html", "Create Account", "Register a parent account on the LEAPS portal."))
		r.Get("/portal", pagesHandler.ServePortal)
		r.Get("/portal/*", pagesHandler.ServePortal)
	})

	// ─── asynq Worker ─────────────────────────────────────────────────────────
	asynqServer := startWorker(cfg, db, emailService, pdfGenerator, cfg.AppBaseURL)
	defer jobClient.Shutdown()

	// ─── HTTP Server ──────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("addr", srv.Addr).Msg("HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	// ─── Graceful Shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Info().Msg("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
	}
	if asynqServer != nil {
		asynqServer.Shutdown()
	}
	log.Info().Msg("server stopped")
}

func startWorker(cfg *config.Config, db *gorm.DB, email *services.EmailService, pdfGen *pdf.Generator, baseURL string) *asynq.Server {
	redisOpt := jobs.ParseRedisOpt(cfg.RedisURL)
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			jobs.QueuePDFGeneration:         6,
			jobs.QueueNotificationsDispatch: 4,
			jobs.QueueQuizTrigger:           8,
			jobs.QueueEmailSend:             2,
		},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			log.Error().Err(err).Str("task_type", task.Type()).Msg("asynq: task failed")
		}),
	})

	mux := asynq.NewServeMux()
	jobs.RegisterHandlers(mux, &jobs.WorkerDeps{
		DB:      db,
		Email:   email,
		PDF:     pdfGen,
		BaseURL: baseURL,
	})

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("asynq worker panicked — recovering")
			}
		}()
		if err := srv.Run(mux); err != nil {
			log.Error().Err(err).Msg("asynq worker error")
		}
	}()

	log.Info().Msg("asynq worker started")
	return srv
}

// extractHost strips scheme and path from a URL to return the bare hostname.
func extractHost(baseURL string) string {
	u := strings.TrimPrefix(baseURL, "http://")
	u = strings.TrimPrefix(u, "https://")
	if i := strings.Index(u, "/"); i >= 0 {
		u = u[:i]
	}
	return u
}

var _ = models.AllModels
