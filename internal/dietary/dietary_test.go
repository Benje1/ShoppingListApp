package dietary

import (
	"reflect"
	"testing"
)

func TestSanitize_NormalisesAndDedupes(t *testing.T) {
	got, err := Sanitize([]string{" Gluten_Free ", "gluten_free", "VEGAN", ""})
	if err != nil {
		t.Fatalf("Sanitize: %v", err)
	}
	want := []string{"gluten_free", "vegan"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Sanitize = %v, want %v", got, want)
	}
}

func TestSanitize_NilReturnsEmptyNonNil(t *testing.T) {
	got, err := Sanitize(nil)
	if err != nil {
		t.Fatalf("Sanitize(nil): %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("Sanitize(nil) = %v, want empty non-nil", got)
	}
}

func TestSanitize_UnknownTagErrors(t *testing.T) {
	if _, err := Sanitize([]string{"gluten_free", "keto"}); err == nil {
		t.Error("expected an error for an unknown dietary tag")
	}
}

func TestOrEmpty_NilBecomesEmpty(t *testing.T) {
	if got := OrEmpty(nil); got == nil || len(got) != 0 {
		t.Errorf("OrEmpty(nil) = %v, want empty non-nil", got)
	}
}
