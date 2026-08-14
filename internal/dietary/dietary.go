// Package dietary holds the fixed set of dietary-variant tags used on meal
// ingredients (e.g. "use the gluten-free pasta on this meal card") together
// with helpers to validate and normalise them.
//
// These are distinct from allergens: an allergen tag says what an item
// *contains*, whereas a dietary tag says which *variant* of an item a meal
// should be made with. Keeping the set in one place stops the backend and the
// two frontends (React web + React Native) drifting apart on valid values.
package dietary

import (
	"fmt"
	"strings"
)

// Allowed is the fixed set of dietary-variant tags a meal ingredient may carry.
var Allowed = map[string]bool{
	"gluten_free": true,
	"dairy_free":  true,
	"vegan":       true,
	"vegetarian":  true,
	"nut_free":    true,
	"egg_free":    true,
}

// Sanitize lowercases, trims, de-duplicates, and drops any empty tag while
// rejecting any tag not in the allowed set. It returns a non-nil (possibly
// empty) slice so the DB column is never NULL and the JSON response is always
// an array. On an unknown tag it returns an error describing the offending tag.
func Sanitize(in []string) ([]string, error) {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, d := range in {
		tag := strings.ToLower(strings.TrimSpace(d))
		if tag == "" || seen[tag] {
			continue
		}
		if !Allowed[tag] {
			return nil, fmt.Errorf("unknown dietary tag %q", d)
		}
		seen[tag] = true
		out = append(out, tag)
	}
	return out, nil
}

// OrEmpty guarantees a non-nil slice so JSON encodes [] rather than null.
func OrEmpty(d []string) []string {
	if d == nil {
		return []string{}
	}
	return d
}
