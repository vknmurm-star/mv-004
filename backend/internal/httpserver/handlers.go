package httpserver

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/timemachine-auto/timemachine/internal/apperror"
	"github.com/timemachine-auto/timemachine/internal/domain/service"
	"github.com/timemachine-auto/timemachine/pkg/webutil"
)

type handlers struct {
	d             *Deps
	seatStore     seatStore
	recentInbound *recentSet
}

func (h *handlers) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.d.DB.Ping(r.Context()); err != nil {
		webutil.WriteError(w, apperror.Internal("db ping failed", err))
		return
	}
	webutil.WriteOK(w, map[string]any{"status": "ok"})
}

// ---------- Public: categories & services ----------

func (h *handlers) ListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.d.Services.ListCategories(r.Context())
	if err != nil {
		webutil.WriteError(w, err)
		return
	}
	webutil.WriteOK(w, cats)
}

func (h *handlers) ListServices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	admin := strings.HasPrefix(r.URL.Path, "/api/v1/admin/")
	params := service.ServiceListParams{
		Search:     q.Get("q"),
		Category:   q.Get("category"),
		Limit:      atoiDefault(q.Get("limit"), 100),
		Offset:     atoiDefault(q.Get("offset"), 0),
		ActiveOnly: !admin,
	}
	list, total, err := h.d.Services.List(r.Context(), params)
	if err != nil {
		webutil.WriteError(w, err)
		return
	}
	webutil.WriteJSON(w, http.StatusOK, webutil.Envelope{
		Data: list,
		Meta: map[string]int{"total": total, "limit": params.Limit, "offset": params.Offset},
	})
}

func (h *handlers) GetService(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		webutil.WriteError(w, apperror.Invalid("invalid id"))
		return
	}
	sv, err := h.d.Services.Get(r.Context(), id)
	if err != nil {
		webutil.WriteError(w, err)
		return
	}
	webutil.WriteOK(w, sv)
}

// ---------- Public: FAQ / reviews ----------

func (h *handlers) ListFAQ(w http.ResponseWriter, r *http.Request) {
	items, err := h.d.FAQ.List(r.Context(), true)
	if err != nil {
		webutil.WriteError(w, apperror.Internal("list faq", err))
		return
	}
	webutil.WriteOK(w, items)
}

func (h *handlers) ListReviews(w http.ResponseWriter, r *http.Request) {
	items, err := h.d.Reviews.List(r.Context(), true)
	if err != nil {
		webutil.WriteError(w, apperror.Internal("list reviews", err))
		return
	}
	webutil.WriteOK(w, items)
}

func (h *handlers) CreateReview(w http.ResponseWriter, r *http.Request) {
	var in reviewInput
	if err := webutil.Decode(r, &in); err != nil {
		webutil.WriteError(w, err)
		return
	}
	if strings.TrimSpace(in.Author) == "" {
		webutil.WriteError(w, apperror.Invalid("укажите имя"))
		return
	}
	id, err := h.d.Reviews.Create(r.Context(), in.reviewRepoInput())
	if err != nil {
		webutil.WriteError(w, apperror.Internal("create review", err))
		return
	}
	webutil.WriteJSON(w, http.StatusCreated, webutil.Envelope{
		Data: map[string]int64{"id": id},
		Meta: map[string]string{"status": "awaiting_moderation"},
	})
}

// ---------- Public: appointments & leads ----------

func (h *handlers) CreateAppointment(w http.ResponseWriter, r *http.Request) {
	var in bookingInput
	if err := webutil.Decode(r, &in); err != nil {
		webutil.WriteError(w, err)
		return
	}
	if err := in.Validate(); err != nil {
		webutil.WriteError(w, err)
		return
	}
	id, err := h.d.Booking.Create(r.Context(), &in.Input)
	if err != nil {
		webutil.WriteError(w, err)
		return
	}

	_ = h.d.Audit.Record(r.Context(), 0, "create", "appointment", strconv.FormatInt(id, 10), in)
	go h.notifyAppointment(id, in)

	webutil.WriteJSON(w, http.StatusCreated, webutil.Envelope{
		Data: map[string]int64{"id": id},
		Meta: map[string]string{"status": "new", "max_notified": boolStr(h.d.Cfg.Max.Enabled)},
	})
}

func (h *handlers) CreateLead(w http.ResponseWriter, r *http.Request) {
	var in leadInput
	if err := webutil.Decode(r, &in); err != nil {
		webutil.WriteError(w, err)
		return
	}
	id, err := h.d.Leads.Create(r.Context(), &in.CreateInput)
	if err != nil {
		webutil.WriteError(w, err)
		return
	}
	webutil.WriteJSON(w, http.StatusCreated, webutil.Envelope{
		Data: map[string]int64{"id": id},
	})
}

func (h *handlers) notifyAppointment(id int64, in bookingInput) {
	ctx, cancel := contextWithTimeout()
	defer cancel()
	_ = h.d.Notifier.NotifyAppointment(ctx, maxAppointmentNotif{
		ID: id, Name: in.Name, Phone: in.Phone,
		CarMake: in.CarMake, CarModel: in.CarModel,
		GovNumber: in.GovNumber, Comment: in.Comment,
	})
}
