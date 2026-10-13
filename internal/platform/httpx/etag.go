package httpx

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/apperr"
)

// ETag formats a resource version as a strong ETag: 3 -> "\"3\"".
func ETag(version int32) string { return `"` + strconv.Itoa(int(version)) + `"` }

// IfMatch parses the If-Match header. ok is false when the header is absent (the update is then
// unconditional, which the contract allows). A malformed header is a 422.
func IfMatch(header string) (version int32, ok bool, err error) {
	h := strings.TrimSpace(header)
	if h == "" {
		return 0, false, nil
	}
	h = strings.TrimPrefix(h, "W/")
	n, convErr := strconv.ParseInt(strings.Trim(h, `"`), 10, 32)
	if convErr != nil || n < 1 {
		return 0, false, apperr.Invalid(apperr.CodeValidation, "If-Match must be an ETag returned by this API.",
			apperr.FieldError{Location: "header.If-Match", Message: "invalid ETag", Value: header})
	}
	return int32(n), true, nil
}

// PageParams is embedded in every list input. Names and bounds match the contract parameters.
type PageParams struct {
	Cursor string `query:"cursor" doc:"nextCursor from the previous page"`
	Limit  int    `query:"limit" minimum:"1" maximum:"200" default:"50"`
}

// EncodeCursor/DecodeCursor make the opaque nextCursor. Use the last row's sort key (usually its
// UUIDv7 id, which sorts by creation time). Clients must treat it as opaque.
func EncodeCursor(v any) *string {
	b, _ := json.Marshal(v)
	s := base64.RawURLEncoding.EncodeToString(b)
	return &s
}

func DecodeCursor(cursor string, into any) error {
	if cursor == "" {
		return nil
	}
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err == nil {
		err = json.Unmarshal(b, into)
	}
	if err != nil {
		return apperr.Invalid(apperr.CodeValidation, "Invalid cursor.",
			apperr.FieldError{Location: "query.cursor", Message: "invalid cursor"})
	}
	return nil
}
