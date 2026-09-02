package category

import "testing"

func TestSanitize_NormalisesCase(t *testing.T) {
	got, err := Sanitize("  Soup ")
	if err != nil {
		t.Fatalf("Sanitize: %v", err)
	}
	if got != "soup" {
		t.Errorf("Sanitize = %q, want %q", got, "soup")
	}
}

func TestSanitize_EmptyIsValid(t *testing.T) {
	got, err := Sanitize("   ")
	if err != nil {
		t.Fatalf("Sanitize(blank): %v", err)
	}
	if got != "" {
		t.Errorf("Sanitize(blank) = %q, want empty", got)
	}
}

func TestSanitize_UnknownErrors(t *testing.T) {
	if _, err := Sanitize("tapas"); err == nil {
		t.Error("expected an error for an unknown category")
	}
}

func TestSanitize_AllAllowedRoundTrip(t *testing.T) {
	for c := range Allowed {
		got, err := Sanitize(c)
		if err != nil {
			t.Errorf("Sanitize(%q): unexpected error %v", c, err)
		}
		if got != c {
			t.Errorf("Sanitize(%q) = %q, want unchanged", c, got)
		}
	}
}
