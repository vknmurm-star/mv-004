package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timemachine-auto/timemachine/internal/aiagent"
	"github.com/timemachine-auto/timemachine/internal/apperror"
	"github.com/timemachine-auto/timemachine/internal/config"
	auditrepo "github.com/timemachine-auto/timemachine/internal/domain/audit"
	authrepo "github.com/timemachine-auto/timemachine/internal/domain/auth"
	bookingrepo "github.com/timemachine-auto/timemachine/internal/domain/booking"
	faqrepo "github.com/timemachine-auto/timemachine/internal/domain/faq"
	kbrepo "github.com/timemachine-auto/timemachine/internal/domain/knowledge"
	leadrepo "github.com/timemachine-auto/timemachine/internal/domain/lead"
	reviewrepo "github.com/timemachine-auto/timemachine/internal/domain/review"
	svcrepo "github.com/timemachine-auto/timemachine/internal/domain/service"
	"github.com/timemachine-auto/timemachine/internal/integrations/max"
	"github.com/timemachine-auto/timemachine/internal/middleware"
	"github.com/timemachine-auto/timemachine/internal/priceimport"
	"github.com/timemachine-auto/timemachine/pkg/webutil"
)

// Deps bundles every dependency the handlers need.
type Deps struct {
	Cfg      *config.Config
	DB       *pgxpool.Pool
	Services *svcrepo.Repo
	Booking  *bookingrepo.Repo
	Leads    *leadrepo.Repo
	FAQ      *faqrepo.Repo
	KB       *kbrepo.Repo
	Reviews  *reviewrepo.Repo
	Auth     *authrepo.Repo
	Audit    *auditrepo.Repo
	Prices   *priceimport.Service
	Max      max.Client
	Notifier *max.Notifier
	MaxStore *max.Store
	Agent    *aiagent.Agent
}

// Router builds the whole public + admin + webhook API tree.
func Router(d *Deps) http.Handler {
	log := d.Cfg.AppEnv

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger(pkgLogger(log)))
	r.Use(middleware.Recover(pkgLogger(log)))
	r.Use(middleware.SecurityHeaders(d.Cfg))
	r.Use(middleware.CORS(d.Cfg.CORSAllowedOrigins))

	apiForm := middleware.NewRateLimiter(float64(d.Cfg.FormRatePerHour)/3600.0, 1)
	apiAPI := middleware.NewRateLimiter(float64(d.Cfg.APIRatePerMin)/60.0, 10)

	h := &handlers{d: d, seatStore: newSeatStore(), recentInbound: newRecentSet(2048)}

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", h.Health)

		// Public content
		r.Group(func(r chi.Router) {
			r.Use(apiAPI.Limit)
			r.Get("/categories", h.ListCategories)
			r.Get("/services", h.ListServices)
			r.Get("/services/{id}", h.GetService)
			r.Get("/faq", h.ListFAQ)
			r.Get("/reviews", h.ListReviews)
		})

		// Lead-generating public endpoints (stricter rate limit per IP)
		r.Group(func(r chi.Router) {
			r.Use(apiForm.Limit)
			r.Post("/appointments", h.CreateAppointment)
			r.Post("/leads", h.CreateLead)
			r.Post("/reviews", h.CreateReview)
		})

		// MAX webhook (no JWT; validated by webhook secret + rate limited)
		webhookRL := middleware.NewRateLimiter(10.0/60.0, 20) // 10/min, burst 20 per IP
		r.Route("/max", func(r chi.Router) {
			r.With(webhookRL.Limit).Post("/webhook", h.MaxWebhook)
			r.Get("/deeplink", h.MaxDeepLink)
		})

		// Admin area
		r.Route("/admin", func(r chi.Router) {
			r.Post("/login", h.AdminLogin)

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAuth(d.Cfg.JWT))
				r.Use(apiAPI.Limit)

				// services / prices
				r.Get("/services", h.AdminListServices)
				r.Post("/services", h.AdminCreateService)
				r.Put("/services/{id}", h.AdminUpdateService)
				r.Delete("/services/{id}", h.AdminDeleteService)

				r.Get("/prices/export", h.AdminExportPrices)
				r.Get("/prices/template", h.AdminPriceTemplate)
				r.Post("/prices/import", h.AdminImportPrices) // parse + create job
				r.Get("/prices/jobs", h.AdminListJobs)
				r.Get("/prices/jobs/{id}", h.AdminGetJob)
				r.Post("/prices/jobs/{id}/apply", h.AdminApplyJob)

				// appointments / leads
				r.Get("/appointments", h.AdminListAppointments)
				r.Patch("/appointments/{id}", h.AdminPatchAppointment)
				r.Get("/leads", h.AdminListLeads)
				r.Patch("/leads/{id}", h.AdminPatchLead)

				// faq / knowledge
				r.Get("/faq", h.AdminListFAQ)
				r.Post("/faq", h.AdminUpsertFAQ)
				r.Put("/faq/{id}", h.AdminUpsertFAQ)
				r.Delete("/faq/{id}", h.AdminDeleteFAQ)

				r.Get("/knowledge", h.AdminListKB)
				r.Post("/knowledge", h.AdminUpsertKB)
				r.Put("/knowledge/{id}", h.AdminUpsertKB)
				r.Delete("/knowledge/{id}", h.AdminDeleteKB)

				r.Get("/reviews", h.AdminListReviews)
				r.Patch("/reviews/{id}", h.AdminPatchReview)

				r.Get("/audit", h.AdminListAudit)
			})
		})
	})

	// OpenAPI document (static)
	r.Get("/api/v1/openapi.yaml", h.OpenAPI)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		webutil.WriteError(w, apperror.NotFound("route not found"))
	})
	return r
}
