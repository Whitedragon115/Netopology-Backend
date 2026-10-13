package pg

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL error codes the services care about. https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	codeUniqueViolation     = "23505"
	codeForeignKeyViolation = "23503"
	codeCheckViolation      = "23514"
	codeRaiseException      = "P0001" // RAISE EXCEPTION in our triggers
)

// IsNotFound reports whether a :one query found no row.
func IsNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// UniqueViolation returns the violated constraint/index name, if err is a unique violation.
// Services translate it, e.g. "workspaces_slug_key" -> apperr.Conflict(apperr.CodeSlugTaken, ...).
func UniqueViolation(err error) (constraint string, ok bool) {
	return constraintFor(err, codeUniqueViolation)
}

// ForeignKeyViolation returns the violated constraint name, if any.
func ForeignKeyViolation(err error) (constraint string, ok bool) {
	return constraintFor(err, codeForeignKeyViolation)
}

// CheckViolation returns the violated CHECK constraint name, if any.
func CheckViolation(err error) (constraint string, ok bool) {
	return constraintFor(err, codeCheckViolation)
}

// TriggerError returns the message of a RAISE EXCEPTION from one of our triggers
// (area cycles, link endpoints not placed, parent placement mismatch, ...).
func TriggerError(err error) (message string, ok bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == codeRaiseException {
		return pgErr.Message, true
	}
	return "", false
}

func constraintFor(err error, code string) (string, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == code {
		return pgErr.ConstraintName, true
	}
	return "", false
}
