// Package allergens holds the fixed set of allergen tags shared across the app
// (meals, shopping items, and sub-ingredients) together with helpers to
// validate and normalise them.
//
// The set is the EU 14 major allergens and is kept in sync with the frontend
// chip list. Keeping it in one place means meals and the shopping catalogue
// can never drift apart on what counts as a valid allergen.
package allergens

import (
	"fmt"
	"strings"
)

// Allowed is the fixed set of allergen tags any entity may carry.
var Allowed = map[string]bool{
	"gluten": true, "crustaceans": true, "eggs": true, "fish": true,
	"peanuts": true, "soy": true, "dairy": true, "nuts": true,
	"celery": true, "mustard": true, "sesame": true, "sulphites": true,
	"lupin": true, "molluscs": true,
}

// Sanitize lowercases, trims, de-duplicates, and drops any empty tag while
// rejecting any tag not in the allowed set. It returns a non-nil (possibly
// empty) slice so the DB column is never NULL and the JSON response is always
// an array. On an unknown tag it returns an error describing the offending tag.
func Sanitize(in []string) ([]string, error) {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, a := range in {
		tag := strings.ToLower(strings.TrimSpace(a))
		if tag == "" || seen[tag] {
			continue
		}
		if !Allowed[tag] {
			return nil, fmt.Errorf("unknown allergen %q", a)
		}
		seen[tag] = true
		out = append(out, tag)
	}
	return out, nil
}

// OrEmpty guarantees a non-nil slice so JSON encodes [] rather than null.
func OrEmpty(a []string) []string {
	if a == nil {
		return []string{}
	}
	return a
}

// Union merges several allergen slices into a single sorted-free, de-duplicated
// slice. The input tags are assumed already sanitised; empty entries are
// skipped. The result is never nil.
func Union(lists ...[]string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0)
	for _, list := range lists {
		for _, a := range list {
			if a == "" || seen[a] {
				continue
			}
			seen[a] = true
			out = append(out, a)
		}
	}
	return out
}
