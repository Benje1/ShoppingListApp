package allergens

import (
	"reflect"
	"testing"
)

func TestSanitize_NormalisesAndDedupes(t *testing.T) {
	got, err := Sanitize([]string{" Gluten ", "gluten", "SOY", ""})
	if err != nil {
		t.Fatalf("Sanitize: %v", err)
	}
	want := []string{"gluten", "soy"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Sanitize = %v, want %v", got, want)
	}
}

func TestSanitize_NilReturnsEmptyNonNil(t *testing.T) {
	got, err := Sanitize(nil)
	if err != nil {
		t.Fatalf("Sanitize(nil): %v", err)
	}
	if got == nil {
		t.Fatal("Sanitize(nil) returned a nil slice; want empty non-nil")
	}
	if len(got) != 0 {
		t.Errorf("Sanitize(nil) = %v, want empty", got)
	}
}

func TestSanitize_UnknownTagErrors(t *testing.T) {
	if _, err := Sanitize([]string{"gluten", "plutonium"}); err == nil {
		t.Error("expected an error for an unknown allergen tag")
	}
}

func TestOrEmpty_NilBecomesEmpty(t *testing.T) {
	if got := OrEmpty(nil); got == nil || len(got) != 0 {
		t.Errorf("OrEmpty(nil) = %v, want empty non-nil", got)
	}
	passthrough := []string{"eggs"}
	if got := OrEmpty(passthrough); !reflect.DeepEqual(got, passthrough) {
		t.Errorf("OrEmpty passthrough = %v, want %v", got, passthrough)
	}
}

func TestUnion_MergesAndDedupes(t *testing.T) {
	got := Union([]string{"gluten", "soy"}, []string{"soy", "eggs"}, nil, []string{""})
	want := []string{"gluten", "soy", "eggs"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Union = %v, want %v", got, want)
	}
}

func TestUnion_EmptyInputIsNonNil(t *testing.T) {
	if got := Union(); got == nil {
		t.Error("Union() returned nil; want empty non-nil slice")
	}
}
