package httpserver

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/timemachine-auto/timemachine/internal/aiagent"
	"github.com/timemachine-auto/timemachine/internal/apperror"
	"github.com/timemachine-auto/timemachine/internal/domain/booking"
	"github.com/timemachine-auto/timemachine/internal/domain/lead"
	"github.com/timemachine-auto/timemachine/internal/domain/review"
	maxpkg "github.com/timemachine-auto/timemachine/internal/integrations/max"
	"github.com/timemachine-auto/timemachine/internal/middleware"
)

// ---------- DTOs wrapping domain inputs ----------

type bookingInput struct {
	booking.Input
}

// reviewInput is the public-create payload.
type reviewInput struct {
	Author  string `json:"author"`
	Rating  int    `json:"rating"`
	Body    string `json:"body"`
	CarInfo string `json:"car_info"`
}

func (in *reviewInput) reviewRepoInput() *review.Input {
	return &review.Input{
		Author:  in.Author,
		Rating:  in.Rating,
		Body:    in.Body,
		CarInfo: in.CarInfo,
	}
}

type leadInput struct {
	lead.CreateInput
}

// leadCreateInput aliases the domain DTO so handlers can build one without
// repeating field names; pointer-to-alias is assignable to the repo method.
type leadCreateInput = lead.CreateInput

// ---------- helpers ----------

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func contextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

type maxAppointmentNotif = maxpkg.AppointmentNotification

func readAllWithLimit(r *http.Request, maxN int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r.Body, maxN))
}

// seatStore is a simple in-memory conversation state cache. For multi-instance
// deployments swap with the sessions table; the agent treats it as opaque.
type seatStore struct {
	data map[string]aiagent.Seat
}

func newSeatStore() seatStore { return seatStore{data: make(map[string]aiagent.Seat)} }

func (s seatStore) load(chatID string) (aiagent.Seat, bool) {
	v, ok := s.data[chatID]
	return v, ok
}

func (s seatStore) save(chatID string, seat aiagent.Seat) { s.data[chatID] = seat }

// openAPIDocument is defined in openapi_embed.go via go:embed.

// maxButtons converts agent button hints into MAX Button payloads.
func maxButtons(a aiagent.Answer) []maxpkg.Button {
	out := make([]maxpkg.Button, 0, len(a.Buttons)+len(a.ButtonLabelHint))
	add := func(labels []string) {
		for _, lb := range labels {
			out = append(out, maxpkg.Button{Label: lb, Payload: strings.ToLower(lb)})
		}
	}
	add(a.Buttons)
	add(a.ButtonLabelHint)
	return out
}

// notUsed keeps the middleware import alive when handlers_admin.go is the only
// consumer of middleware.UserID.
var _ = middleware.CtxEmail

// errInternalf returns a formatted internal error.
func errInternalf(format string, a ...any) error {
	return apperror.Internal(format, nil)
}
