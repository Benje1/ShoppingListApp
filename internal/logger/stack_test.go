package logger

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestWithStack_NilReturnsNil(t *testing.T) {
	if WithStack(nil) != nil {
		t.Error("expected WithStack(nil) to return nil")
	}
}

func TestWithStack_WrapsError(t *testing.T) {
	base := errors.New("boom")
	wrapped := WithStack(base)

	if wrapped == nil {
		t.Fatal("expected non-nil wrapped error")
	}
	if wrapped.Error() != "boom" {
		t.Errorf("expected message 'boom', got %q", wrapped.Error())
	}
}

func TestWithStack_UnwrapsToOriginal(t *testing.T) {
	base := errors.New("original")
	wrapped := WithStack(base)

	if !errors.Is(wrapped, base) {
		t.Error("errors.Is should find the original error through the wrapper")
	}
}

func TestWithStack_PreservesExistingStackError(t *testing.T) {
	base := errors.New("base")
	first := WithStack(base)
	second := WithStack(first)

	if first != second {
		t.Error("wrapping a StackError a second time should return the same pointer")
	}
}

func TestStackTrace_ContainsCallerFunction(t *testing.T) {
	err := WithStack(errors.New("trace me"))
	se := err.(*StackError)
	trace := se.StackTrace()

	if !strings.Contains(trace, "TestStackTrace_ContainsCallerFunction") {
		t.Errorf("expected stack trace to contain the test function name, got:\n%s", trace)
	}
}

func TestStackTrace_ContainsFileAndLine(t *testing.T) {
	err := WithStack(errors.New("trace me"))
	se := err.(*StackError)
	trace := se.StackTrace()

	if !strings.Contains(trace, "stack_test.go") {
		t.Errorf("expected stack trace to contain 'stack_test.go', got:\n%s", trace)
	}
}

func TestFindStackError_FindsThroughChain(t *testing.T) {
	base := errors.New("root")
	withStack := WithStack(base)
	wrapped := fmt.Errorf("outer: %w", withStack) //nolint

	se := findStackError(wrapped)
	if se == nil {
		t.Fatal("expected findStackError to find StackError through error chain")
	}
}

func TestAppendStackIfPresent_AddsStackKey(t *testing.T) {
	err := WithStack(errors.New("with stack"))
	args := []any{"error", err}
	result := appendStackIfPresent(args)

	// Should have added "stack" key and trace value
	if len(result) != 4 {
		t.Fatalf("expected 4 args (original 2 + stack key/value), got %d", len(result))
	}
	if result[2] != "stack" {
		t.Errorf("expected third arg to be 'stack', got %v", result[2])
	}
	trace, ok := result[3].(string)
	if !ok || trace == "" {
		t.Error("expected fourth arg to be a non-empty stack trace string")
	}
}

func TestAppendStackIfPresent_NoOpForPlainError(t *testing.T) {
	err := errors.New("plain error")
	args := []any{"error", err}
	result := appendStackIfPresent(args)

	if len(result) != 2 {
		t.Errorf("expected args unchanged for plain error, got %d args", len(result))
	}
}

func TestAppendStackIfPresent_NoOpForNoError(t *testing.T) {
	args := []any{"key", "value"}
	result := appendStackIfPresent(args)

	if len(result) != 2 {
		t.Errorf("expected args unchanged when no error present, got %d args", len(result))
	}
}
