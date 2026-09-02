// Package category holds the fixed set of meal categories together with helpers
// to validate and normalise them.
//
// A category is the "slot type" a meal fills when planning the week: you plan a
// "soup night", and any meal in the `soup` category is interchangeable in that
// slot. Keeping the set in one place stops the backend and the two frontends
// (React web + React Native) drifting apart on valid values.
//
// This mirrors the internal/dietary and internal/allergens packages, but a meal
// carries exactly one category (a single string, not a set), and the empty
// string is a valid value meaning "uncategorised".
package category

import (
	"fmt"
	"strings"
)

// Allowed is the fixed set of meal categories. Keep this in sync with
// MEAL_CATEGORIES / CATEGORY_META in the web app's src/components/Categories.tsx.
var Allowed = map[string]bool{
	"soup":      true,
	"salad":     true,
	"pasta":     true,
	"curry":     true,
	"stir_fry":  true,
	"roast":     true,
	"grill":     true,
	"pie":       true,
	"stew":      true,
	"sandwich":  true,
	"breakfast": true,
	"dessert":   true,
	"side":      true,
}

// Sanitize trims and lowercases the category, then validates it against the
// allowed set. The empty string is valid and means "uncategorised". Any other
// value not in the set returns an error naming the offending value (surfaced to
// the caller as an HTTP 400).
func Sanitize(in string) (string, error) {
	c := strings.ToLower(strings.TrimSpace(in))
	if c == "" {
		return "", nil
	}
	if !Allowed[c] {
		return "", fmt.Errorf("unknown category %q", in)
	}
	return c, nil
}
