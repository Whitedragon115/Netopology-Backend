// Package slug makes URL slugs that satisfy workspaces.slug's CHECK (^[a-z0-9][a-z0-9-]{0,62}$).
package slug

import "strings"

// Make turns a display name into a slug: "Home Lab #2" -> "home-lab-2". Never returns "".
// Callers still handle the unique violation (append "-2", "-3", ... or return slug.taken).
func Make(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
		if b.Len() >= 63 {
			break
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > 63 {
		s = strings.Trim(s[:63], "-")
	}
	if s == "" {
		return "workspace"
	}
	return s
}
