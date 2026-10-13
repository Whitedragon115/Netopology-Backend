// Package apperr defines the typed errors that services return.
//
// Services never know about HTTP. They return an *Error with a Kind and a stable Code
// (for example "inventory.used_elsewhere"); internal/platform/httpx maps the Kind to a status
// code and the Code into the RFC 9457 problem body the frontend switches on.
//
// Codes are part of the API contract: never rename one, only add new ones.
package apperr

import (
	"errors"
	"fmt"
)

// Kind is the class of failure. httpx maps each Kind to exactly one HTTP status.
type Kind int

const (
	KindInternal           Kind = iota // 500, never shows Detail to the client
	KindInvalid                        // 422 validation failed
	KindUnauthenticated                // 401 not signed in / bad credentials / bad share password
	KindForbidden                      // 403 member but not allowed (never for non-members: use KindNotFound)
	KindNotFound                       // 404 missing, or the caller may not know it exists
	KindConflict                       // 409 conflicts with current state
	KindPreconditionFailed             // 412 stale If-Match
	KindLocked                         // 423 canvas locked
	KindRateLimited                    // 429
)

// Error is the only error type a service should return for an expected failure.
// Anything else is treated as a 500 and logged.
type Error struct {
	Kind   Kind
	Code   string // stable machine-readable code, e.g. "member.last_owner"
	Detail string // human-readable, safe to show to the caller
	// Fields holds field-level validation problems (KindInvalid only).
	Fields []FieldError
	// RetryAfter is in seconds (KindRateLimited only).
	RetryAfter int
	cause      error
}

// FieldError is one invalid input, e.g. {Location: "body.name", Message: "too long"}.
type FieldError struct {
	Location string
	Message  string
	Value    any
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Detail, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

func (e *Error) Unwrap() error { return e.cause }

// Wrap attaches an underlying cause (logged, never sent to the client).
func (e *Error) Wrap(cause error) *Error { e.cause = cause; return e }

// As returns the *Error inside err, if any.
func As(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}

// IsKind reports whether err is an *Error of the given kind.
func IsKind(err error, k Kind) bool {
	e, ok := As(err)
	return ok && e.Kind == k
}

func newErr(k Kind, code, detail string) *Error { return &Error{Kind: k, Code: code, Detail: detail} }

func Invalid(code, detail string, fields ...FieldError) *Error {
	e := newErr(KindInvalid, code, detail)
	e.Fields = fields
	return e
}

func Unauthenticated(code, detail string) *Error { return newErr(KindUnauthenticated, code, detail) }
func Forbidden(code, detail string) *Error       { return newErr(KindForbidden, code, detail) }
func NotFound(code, detail string) *Error        { return newErr(KindNotFound, code, detail) }
func Conflict(code, detail string) *Error        { return newErr(KindConflict, code, detail) }
func Locked(code, detail string) *Error          { return newErr(KindLocked, code, detail) }
func PreconditionFailed() *Error {
	return newErr(KindPreconditionFailed, "version.stale", "Someone else changed this first. Reload and try again.")
}

func RateLimited(retryAfter int) *Error {
	e := newErr(KindRateLimited, "rate_limited", "Too many attempts. Try again later.")
	e.RetryAfter = retryAfter
	return e
}

// Common codes used by more than one feature. Feature-specific codes live next to the feature.
const (
	CodeNotFound          = "not_found"
	CodeRoleInsufficient  = "role.insufficient"
	CodeProtected         = "protected"
	CodeNotSignedIn       = "auth.required"
	CodeValidation        = "validation"
	CodeSlugTaken         = "slug.taken"
	CodeCanvasLocked      = "canvas.locked"
	CodeUsedElsewhere     = "inventory.used_elsewhere"
	CodePlaceOnRequired   = "inventory.place_on_required"
	CodeLastOwner         = "member.last_owner"
	CodeSetupDone         = "setup.already_completed"
	CodeIdentityNotLinked = "auth.identity_not_linked"
	CodeLastSignInMethod  = "auth.last_sign_in_method"
)

// ErrNotFound is the generic 404. Use it for "does not exist" AND "you may not know it exists".
func ErrNotFound() *Error { return NotFound(CodeNotFound, "Not found.") }

// ErrRoleInsufficient is the generic 403 for a member whose role lacks the permission.
func ErrRoleInsufficient() *Error {
	return Forbidden(CodeRoleInsufficient, "Your role in this workspace does not allow this.")
}
