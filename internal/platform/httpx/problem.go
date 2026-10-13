// Package httpx is the HTTP glue shared by every feature: problem+json errors, operation
// definitions, middleware, cookies, ETags and pagination.
package httpx

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
)

// Problem is the RFC 9457 body of every error response (contract schema "Problem").
// The Go type name matters: Huma names the OpenAPI schema after it.
type Problem struct {
	Type     string         `json:"type,omitempty" format:"uri" default:"about:blank"`
	Title    string         `json:"title"`
	Status   int            `json:"status"`
	Code     string         `json:"code" doc:"Stable machine-readable reason"`
	Detail   string         `json:"detail,omitempty"`
	Instance string         `json:"instance,omitempty"`
	Errors   []ProblemField `json:"errors,omitempty" doc:"Field-level validation errors"`

	headers http.Header
}

type ProblemField struct {
	Location string `json:"location,omitempty"`
	Message  string `json:"message"`
	Value    any    `json:"value,omitempty"`
}

func (p *Problem) Error() string                { return p.Code + ": " + p.Detail }
func (p *Problem) GetStatus() int               { return p.Status }
func (p *Problem) GetHeaders() http.Header      { return p.headers }
func (p *Problem) ContentType(ct string) string { return "application/problem+json" }

var statusForKind = map[apperr.Kind]int{
	apperr.KindInternal:           http.StatusInternalServerError,
	apperr.KindInvalid:            http.StatusUnprocessableEntity,
	apperr.KindUnauthenticated:    http.StatusUnauthorized,
	apperr.KindForbidden:          http.StatusForbidden,
	apperr.KindNotFound:           http.StatusNotFound,
	apperr.KindConflict:           http.StatusConflict,
	apperr.KindPreconditionFailed: http.StatusPreconditionFailed,
	apperr.KindLocked:             http.StatusLocked,
	apperr.KindRateLimited:        http.StatusTooManyRequests,
}

// InstallErrors makes Huma produce our Problem for every error, including the *apperr.Error a
// handler returns. So handlers just `return nil, err` with whatever the service returned.
// Huma's error constructors are package globals, so this is called once from main.
func InstallErrors(log *slog.Logger) {
	huma.NewErrorWithContext = func(hctx huma.Context, status int, msg string, errs ...error) huma.StatusError {
		for _, err := range errs {
			if ae, ok := apperr.As(err); ok {
				if ae.Kind == apperr.KindInternal {
					break
				}
				p := fromApp(ae)
				if ae.RetryAfter > 0 && hctx != nil {
					hctx.SetHeader("Retry-After", strconv.Itoa(ae.RetryAfter))
				}
				return p
			}
		}
		if status >= 500 {
			// never leak internals; log them with the request's context attributes
			if hctx != nil {
				log.ErrorContext(hctx.Context(), "request failed", "status", status, "msg", msg, "errors", errs)
			} else {
				log.Error("request failed", "status", status, "msg", msg, "errors", errs)
			}
			return &Problem{
				Status: http.StatusInternalServerError, Title: http.StatusText(http.StatusInternalServerError), Code: "internal",
				Detail: "Something went wrong on the server. It has been logged.",
			}
		}
		// Huma's own errors: request validation (422), malformed body (400), etc.
		p := &Problem{Status: status, Title: http.StatusText(status), Detail: msg, Code: codeForStatus(status)}
		for _, err := range errs {
			if d, ok := err.(huma.ErrorDetailer); ok {
				det := d.ErrorDetail()
				p.Errors = append(p.Errors, ProblemField{Location: det.Location, Message: det.Message, Value: det.Value})
			} else if err != nil {
				p.Errors = append(p.Errors, ProblemField{Message: err.Error()})
			}
		}
		return p
	}
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		return huma.NewErrorWithContext(nil, status, msg, errs...)
	}
}

func fromApp(e *apperr.Error) *Problem {
	status := statusForKind[e.Kind]
	p := &Problem{Status: status, Title: http.StatusText(status), Code: e.Code, Detail: e.Detail}
	for _, f := range e.Fields {
		p.Errors = append(p.Errors, ProblemField{Location: f.Location, Message: f.Message, Value: f.Value})
	}
	return p
}

func codeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnprocessableEntity:
		return apperr.CodeValidation
	case http.StatusNotFound:
		return apperr.CodeNotFound
	case http.StatusMethodNotAllowed:
		return "method_not_allowed"
	case http.StatusRequestEntityTooLarge:
		return "body_too_large"
	default:
		return "http_" + strconv.Itoa(status)
	}
}
