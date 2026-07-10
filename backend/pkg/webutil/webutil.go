package webutil

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/timemachine-auto/timemachine/internal/apperror"
)

// Envelope is the canonical JSON API envelope.
type Envelope struct {
	Data  any `json:"data,omitempty"`
	Error any `json:"error,omitempty"`
	Meta  any `json:"meta,omitempty"`
}

// ValidationError carries field-level problems.
type ValidationError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (v *ValidationError) Error() string { return v.Message }

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteOK(w http.ResponseWriter, v any) { WriteJSON(w, http.StatusOK, v) }

func WriteError(w http.ResponseWriter, err error) {
	var ve *ValidationError
	if errors.As(err, &ve) {
		data := map[string]any{"code": ve.Code, "message": ve.Message}
		if len(ve.Fields) > 0 {
			data["fields"] = ve.Fields
		}
		WriteJSON(w, http.StatusBadRequest, Envelope{Error: data})
		return
	}
	status := apperror.StatusOf(err)
	WriteJSON(w, status, Envelope{
		Error: map[string]string{
			"code":    string(apperror.AsCode(err)),
			"message": publicMessage(err),
		},
	})
}

// publicMessage hides internal details in non-debug builds.
func publicMessage(err error) string {
	var ae *apperror.Error
	if errors.As(err, &ae) {
		return ae.Message
	}
	return "internal error"
}

const maxJSONBody = 1 << 20 // 1 MiB

// Decode reads JSON into dst and returns a validation-like error on failure.
// The body is size-capped to protect against trivial memory-exhaustion DoS.
func Decode(r *http.Request, dst any) error {
	if r.Body == nil {
		return apperror.Invalid("request body is required")
	}
	r.Body = http.MaxBytesReader(nil, r.Body, maxJSONBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return apperror.Invalid("malformed JSON: " + err.Error())
	}
	return nil
}
