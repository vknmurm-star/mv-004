package httpserver

import (
	"bytes"
	"encoding/csv"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/timemachine-auto/timemachine/internal/apperror"
	"github.com/timemachine-auto/timemachine/internal/middleware"
	"github.com/timemachine-auto/timemachine/internal/priceimport"
	"github.com/timemachine-auto/timemachine/pkg/webutil"
)

// ---------- Auth ----------

func (h *handlers) AdminLogin(w http.ResponseWriter, r *http.Request) {
	var in loginRepoInput
	if err := webutil.Decode(r, &in); err != nil {
		webutil.WriteError(w, err)
		return
	}
	token, u, err := h.d.Auth.IssueToken(r.Context(), &in, h.d.Cfg.JWT)
	if err != nil {
		webutil.WriteError(w, err)
		return
	}
	webutil.WriteOK(w, map[string]any{
		"token":      token,
		"user":       map[string]any{"id": u.ID, "email": u.Email, "role": u.Role, "name": u.Name},
		"expires_in": int(h.d.Cfg.JWT.TTL.Seconds()),
	})
}

// ---------- Services (admin) ----------

func (h *handlers) AdminListServices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := paramsFromQuery(q, false)
	list, total, err := h.d.Services.List(r.Context(), params)
	if err != nil {
		webutil.WriteError(w, err)
		return
	}
	webutil.WriteJSON(w, http.StatusOK, webutil.Envelope{
		Data: list, Meta: map[string]int{"total": total, "limit": params.Limit, "offset": params.Offset},
	})
}

func (h *handlers) AdminCreateService(w http.ResponseWriter, r *http.Request) {
	var in serviceInput
	if err := webutil.Decode(r, &in); err != nil {
		webutil.WriteError(w, err)
		return
	}
	repoIn := in.toRepoInput()
	if err := repoIn.Validate(); err != nil {
		webutil.WriteError(w, err)
		return
	}
	id, err := h.d.Services.Create(r.Context(), repoIn)
	if err != nil {
		webutil.WriteError(w, err)
		return
	}
	_ = h.d.Audit.Record(r.Context(), userIDInt(r), "create", "service", strconv.FormatInt(id, 10), in)
	webutil.WriteJSON(w, http.StatusCreated, webutil.Envelope{Data: map[string]int64{"id": id}})
}

func (h *handlers) AdminUpdateService(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		webutil.WriteError(w, apperror.Invalid("invalid id"))
		return
	}
	var in serviceInput
	if err := webutil.Decode(r, &in); err != nil {
		webutil.WriteError(w, err)
		return
	}
	repoIn := in.toRepoInput()
	if err := repoIn.Validate(); err != nil {
		webutil.WriteError(w, err)
		return
	}
	if err := h.d.Services.Update(r.Context(), id, repoIn); err != nil {
		webutil.WriteError(w, err)
		return
	}
	_ = h.d.Audit.Record(r.Context(), userIDInt(r), "update", "service", strconv.FormatInt(id, 10), in)
	webutil.WriteOK(w, map[string]string{"status": "updated"})
}

func (h *handlers) AdminDeleteService(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		webutil.WriteError(w, apperror.Invalid("invalid id"))
		return
	}
	if err := h.d.Services.Delete(r.Context(), id); err != nil {
		webutil.WriteError(w, err)
		return
	}
	_ = h.d.Audit.Record(r.Context(), userIDInt(r), "delete", "service", strconv.FormatInt(id, 10), nil)
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Prices: template / export / import / jobs ----------

func (h *handlers) AdminPriceTemplate(w http.ResponseWriter, r *http.Request) {
	writeCSVAttachment(w, "timemachine-price-template.csv", priceimport.Template())
}

func (h *handlers) AdminExportPrices(w http.ResponseWriter, r *http.Request) {
	b, err := h.d.Prices.Export(r.Context())
	if err != nil {
		webutil.WriteError(w, err)
		return
	}
	writeCSVAttachment(w, "timemachine-prices.csv", b)
}

func (h *handlers) AdminImportPrices(w http.ResponseWriter, r *http.Request) {
	// 5 MiB hard cap on uploads.
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		webutil.WriteError(w, apperror.Invalid("file too large or malformed: "+err.Error()))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		webutil.WriteError(w, apperror.Invalid("'file' field is required"))
		return
	}
	defer file.Close()

	format := priceimport.DetectFormat(header.Filename)
	rows, summary, err := h.d.Prices.Parse(file, format, header.Filename)
	if err != nil {
		webutil.WriteError(w, apperror.Unprocessable("импорт не разобран: "+err.Error()))
		return
	}

	uid := userIDInt(r)
	jobID, err := h.d.Prices.CreateJob(r.Context(), header.Filename, uid, rows, summary)
	if err != nil {
		webutil.WriteError(w, apperror.Internal("create import job", err))
		return
	}
	_ = h.d.Audit.Record(r.Context(), uid, "import.create", "price_import_job", strconv.FormatInt(jobID, 10), summary)

	webutil.WriteJSON(w, http.StatusCreated, webutil.Envelope{
		Data: map[string]int64{"job_id": jobID},
		Meta: map[string]any{
			"summary":      summary,
			"row_count":    len(rows),
			"preview_rows": previewRows(rows, 50),
			"file_name":    header.Filename,
			"format":       string(format),
			"next":         "POST /api/v1/admin/prices/jobs/{id}/apply",
		},
	})
}

func (h *handlers) AdminListJobs(w http.ResponseWriter, r *http.Request) {
	rows, err := h.d.DB.Query(r.Context(), `
		SELECT id, status, file_name, total_rows, added, updated, skipped,
		COALESCE(to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),''),
		COALESCE(to_char(applied_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),'')
		FROM price_import_jobs ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		webutil.WriteError(w, apperror.Internal("list jobs", err))
		return
	}
	defer rows.Close()
	out := make([]map[string]any, 0)
	for rows.Next() {
		var id int64
		var status, fname, created, applied string
		var total, added, updated, skipped int
		if err := rows.Scan(&id, &status, &fname, &total, &added, &updated, &skipped, &created, &applied); err != nil {
			webutil.WriteError(w, apperror.Internal("scan job", err))
			return
		}
		out = append(out, map[string]any{
			"id": id, "status": status, "file_name": fname,
			"total": total, "added": added, "updated": updated, "skipped": skipped,
			"created_at": created, "applied_at": applied,
		})
	}
	webutil.WriteOK(w, out)
}

func (h *handlers) AdminGetJob(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		webutil.WriteError(w, apperror.Invalid("invalid id"))
		return
	}
	rows, err := h.d.DB.Query(r.Context(), `
		SELECT row_number, action, payload, errors
		FROM price_import_rows WHERE job_id=$1 ORDER BY row_number`, id)
	if err != nil {
		webutil.WriteError(w, apperror.Internal("load job rows", err))
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var n int
		var action string
		var payload, errjson []byte
		if err := rows.Scan(&n, &action, &payload, &errjson); err != nil {
			webutil.WriteError(w, apperror.Internal("scan row", err))
			return
		}
		items = append(items, map[string]any{
			"row_number": n, "action": action, "payload": jsonRaw(payload), "errors": jsonRaw(errjson),
		})
	}
	webutil.WriteOK(w, map[string]any{"job_id": id, "rows": items})
}

func (h *handlers) AdminApplyJob(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		webutil.WriteError(w, apperror.Invalid("invalid id"))
		return
	}
	sum, err := h.d.Prices.Apply(r.Context(), id)
	if err != nil {
		webutil.WriteError(w, apperror.Internal("apply import", err))
		return
	}
	_ = h.d.Audit.Record(r.Context(), userIDInt(r), "import.apply", "price_import_job", strconv.FormatInt(id, 10), sum)
	webutil.WriteOK(w, map[string]any{"job_id": id, "summary": sum})
}

// ---------- Appointments / leads (admin) ----------

func (h *handlers) AdminListAppointments(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := atoiDefault(r.URL.Query().Get("limit"), 100)
	list, total, err := h.d.Booking.List(r.Context(), bookingRepoListParams{Status: status, Limit: limit})
	if err != nil {
		webutil.WriteError(w, err)
		return
	}
	webutil.WriteJSON(w, http.StatusOK, webutil.Envelope{
		Data: list, Meta: map[string]int{"total": total, "limit": limit},
	})
}

func (h *handlers) AdminPatchAppointment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		webutil.WriteError(w, apperror.Invalid("invalid id"))
		return
	}
	var in patchAppointment
	if err := webutil.Decode(r, &in); err != nil {
		webutil.WriteError(w, err)
		return
	}
	if in.Status == "" && in.ManagerNote == "" {
		webutil.WriteError(w, apperror.Invalid("nothing to update"))
		return
	}
	if err := h.d.Booking.SetStatus(r.Context(), id, in.Status, in.ManagerNote); err != nil {
		webutil.WriteError(w, err)
		return
	}
	_ = h.d.Audit.Record(r.Context(), userIDInt(r), "patch", "appointment", strconv.FormatInt(id, 10), in)
	webutil.WriteOK(w, map[string]string{"status": "updated"})
}

func (h *handlers) AdminListLeads(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := atoiDefault(r.URL.Query().Get("limit"), 100)
	list, total, err := h.d.Leads.List(r.Context(), status, limit)
	if err != nil {
		webutil.WriteError(w, err)
		return
	}
	webutil.WriteJSON(w, http.StatusOK, webutil.Envelope{
		Data: list, Meta: map[string]int{"total": total, "limit": limit},
	})
}

func (h *handlers) AdminPatchLead(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		webutil.WriteError(w, apperror.Invalid("invalid id"))
		return
	}
	var in patchLead
	if err := webutil.Decode(r, &in); err != nil {
		webutil.WriteError(w, err)
		return
	}
	if err := h.d.Leads.SetStatus(r.Context(), id, in.Status); err != nil {
		webutil.WriteError(w, err)
		return
	}
	_ = h.d.Audit.Record(r.Context(), userIDInt(r), "patch", "lead", strconv.FormatInt(id, 10), in)
	webutil.WriteOK(w, map[string]string{"status": "updated"})
}

// ---------- FAQ (admin) ----------

func (h *handlers) AdminListFAQ(w http.ResponseWriter, r *http.Request) {
	items, err := h.d.FAQ.List(r.Context(), false)
	if err != nil {
		webutil.WriteError(w, apperror.Internal("list faq", err))
		return
	}
	webutil.WriteOK(w, items)
}

func (h *handlers) AdminUpsertFAQ(w http.ResponseWriter, r *http.Request) {
	var in faqInput
	if err := webutil.Decode(r, &in); err != nil {
		webutil.WriteError(w, err)
		return
	}
	if strings.TrimSpace(in.Question) == "" || strings.TrimSpace(in.Answer) == "" {
		webutil.WriteError(w, apperror.Invalid("question и answer обязательны"))
		return
	}
	var id int64
	if pid := chi.URLParam(r, "id"); pid != "" {
		n, _ := strconv.ParseInt(pid, 10, 64)
		if err := h.d.FAQ.Update(r.Context(), n, toFaqRepoInput(in)); err != nil {
			webutil.WriteError(w, err)
			return
		}
		id = n
	} else {
		created, err := h.d.FAQ.Create(r.Context(), toFaqRepoInput(in))
		if err != nil {
			webutil.WriteError(w, err)
			return
		}
		id = created
	}
	_ = h.d.Audit.Record(r.Context(), userIDInt(r), "upsert", "faq", strconv.FormatInt(id, 10), in)
	webutil.WriteOK(w, map[string]int64{"id": id})
}

func (h *handlers) AdminDeleteFAQ(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		webutil.WriteError(w, apperror.Invalid("invalid id"))
		return
	}
	_ = h.d.FAQ.Delete(r.Context(), id)
	_ = h.d.Audit.Record(r.Context(), userIDInt(r), "delete", "faq", strconv.FormatInt(id, 10), nil)
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Knowledge (admin) ----------

func (h *handlers) AdminListKB(w http.ResponseWriter, r *http.Request) {
	items, err := h.d.KB.List(r.Context())
	if err != nil {
		webutil.WriteError(w, err)
		return
	}
	webutil.WriteOK(w, items)
}

func (h *handlers) AdminUpsertKB(w http.ResponseWriter, r *http.Request) {
	var in knowledgeInput
	if err := webutil.Decode(r, &in); err != nil {
		webutil.WriteError(w, err)
		return
	}
	var id int64
	if pid := chi.URLParam(r, "id"); pid != "" {
		n, _ := strconv.ParseInt(pid, 10, 64)
		id = n
	} else {
		id = 0
	}
	saved, err := h.d.KB.Upsert(r.Context(), id, in.toRepoInput(), userIDInt(r))
	if err != nil {
		webutil.WriteError(w, err)
		return
	}
	_ = h.d.Audit.Record(r.Context(), userIDInt(r), "upsert", "knowledge", strconv.FormatInt(saved, 10), in)
	webutil.WriteOK(w, map[string]int64{"id": saved})
}

func (h *handlers) AdminDeleteKB(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		webutil.WriteError(w, apperror.Invalid("invalid id"))
		return
	}
	_ = h.d.KB.Delete(r.Context(), id)
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Reviews (admin) ----------

func (h *handlers) AdminListReviews(w http.ResponseWriter, r *http.Request) {
	items, err := h.d.Reviews.List(r.Context(), false)
	if err != nil {
		webutil.WriteError(w, apperror.Internal("list reviews", err))
		return
	}
	webutil.WriteOK(w, items)
}

func (h *handlers) AdminPatchReview(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		webutil.WriteError(w, apperror.Invalid("invalid id"))
		return
	}
	var in patchReview
	if err := webutil.Decode(r, &in); err != nil {
		webutil.WriteError(w, err)
		return
	}
	if err := h.d.Reviews.SetPublished(r.Context(), id, in.IsPublished); err != nil {
		webutil.WriteError(w, err)
		return
	}
	webutil.WriteOK(w, map[string]string{"status": "updated"})
}

// ---------- Audit ----------

func (h *handlers) AdminListAudit(w http.ResponseWriter, r *http.Request) {
	limit := atoiDefault(r.URL.Query().Get("limit"), 50)
	offset := atoiDefault(r.URL.Query().Get("offset"), 0)
	entries, err := h.d.Audit.List(r.Context(), limit, offset)
	if err != nil {
		webutil.WriteError(w, apperror.Internal("list audit", err))
		return
	}
	webutil.WriteOK(w, entries)
}

// ---------- small DTO + helpers ----------

type serviceInput struct {
	CategoryID      int64  `json:"category_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	LongDescription string `json:"long_description"`
	PriceCents      int64  `json:"price_cents"`
	Currency        string `json:"currency"`
	DurationMin     int    `json:"duration_minutes"`
	IsFromPrice     bool   `json:"is_from_price"`
	IsActive        bool   `json:"is_active"`
	SortOrder       int    `json:"sort_order"`
}

func (in *serviceInput) toRepoInput() *serviceRepoInput {
	return &serviceRepoInput{
		CategoryID:      in.CategoryID,
		Name:            in.Name,
		Description:     in.Description,
		LongDescription: in.LongDescription,
		PriceCents:      in.PriceCents,
		Currency:        in.Currency,
		DurationMin:     in.DurationMin,
		IsFromPrice:     in.IsFromPrice,
		IsActive:        in.IsActive,
		SortOrder:       in.SortOrder,
	}
}

type patchAppointment struct {
	Status      string `json:"status"`
	ManagerNote string `json:"manager_note"`
}
type patchLead struct {
	Status string `json:"status"`
}
type patchReview struct {
	IsPublished bool `json:"is_published"`
}
type faqInput struct {
	Question    string `json:"question"`
	Answer      string `json:"answer"`
	Category    string `json:"category"`
	SortOrder   int    `json:"sort_order"`
	IsPublished bool   `json:"is_published"`
}
type knowledgeInput struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Tags   []string `json:"tags"`
	Source string   `json:"source"`
}

func (in *knowledgeInput) toRepoInput() *knowledgeRepoInput {
	return &knowledgeRepoInput{Title: in.Title, Body: in.Body, Tags: in.Tags, Source: in.Source}
}

func writeCSVAttachment(w http.ResponseWriter, name string, data []byte) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = w.Write(data)
}

// previewRows trims to maxRows and drops payloads to keep the preview response small.
func previewRows(rows []priceimport.ParsedRow, maxRows int) []map[string]any {
	if maxRows <= 0 || maxRows > len(rows) {
		maxRows = len(rows)
	}
	out := make([]map[string]any, 0, maxRows)
	for i := 0; i < maxRows; i++ {
		r := rows[i]
		out = append(out, map[string]any{
			"row_number":  r.RowNumber,
			"action":      r.Action,
			"name":        r.Payload.Name,
			"category_id": r.Payload.CategoryID,
			"price_cents": r.Payload.PriceCents,
			"errors":      r.Errors,
		})
	}
	return out
}

// jsonRaw is a byte slice that marshals as raw JSON instead of base64 string.
type jsonRaw []byte

func (r jsonRaw) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return r, nil
}

func userIDInt(r *http.Request) int64 {
	return middleware.UserID(r)
}

// paramsFromQuery builds the shared list params for admin endpoints.
func paramsFromQuery(q urlQuery, activeOnly bool) serviceRepoListParams {
	return serviceRepoListParams{
		Search:     q.Get("q"),
		Category:   q.Get("category"),
		Limit:      atoiDefault(q.Get("limit"), 100),
		Offset:     atoiDefault(q.Get("offset"), 0),
		ActiveOnly: activeOnly,
	}
}

// csvBoot avoids unused imports when writeCSVAttachment is the only csv user.
var _ = csv.NewWriter
var _ = bytes.NewBuffer
var _ = time.Second
var _ = io.EOF
