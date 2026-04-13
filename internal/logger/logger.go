// Package logger provides a structured logger (slog) that writes to both
// stdout and a daily-rotating file under the configured log directory.
//
// Usage:
//
//	// In main, before anything else:
//	if err := logger.Init("logs"); err != nil {
//	    panic(err)
//	}
//
//	// Everywhere else:
//	logger.Info("server started", "port", 8080)
//	logger.Error("db query failed", "err", err, "query", "GetUser")
//	logger.Warn("non-fatal issue", "detail", "pantry sync skipped")
package logger

import (
	"io"
	"log/slog"
	"os"
)

// L is the package-level logger. Call Init before using it; it falls back
// to a stderr-only logger if Init has not been called.
var L *slog.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))

// Init sets up the package-level logger to write structured text to both
// stdout and a daily-rotating file under logDir (e.g. "logs").
// Call this once at program startup before any other code uses the logger.
func Init(logDir string) error {
	fileWriter, err := newRotatingWriter(logDir)
	if err != nil {
		return err
	}

	// Fan out: write every log line to both stdout and the file.
	multi := io.MultiWriter(os.Stdout, fileWriter)

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		// Include the source file and line in every record so errors are
		// trivially traceable without a stack trace.
		AddSource: true,
	}

	L = slog.New(slog.NewTextHandler(multi, opts))

	// Also set the default slog logger so any third-party code that calls
	// slog.Info / slog.Error goes to the same destination.
	slog.SetDefault(L)

	return nil
}

// ── Convenience wrappers ──────────────────────────────────────────────────────
// These mirror the slog top-level functions so callers don't need to import
// both packages.

func Debug(msg string, args ...any) { L.Debug(msg, args...) }
func Info(msg string, args ...any)  { L.Info(msg, args...) }
func Warn(msg string, args ...any)  { L.Warn(msg, args...) }
func Error(msg string, args ...any) { L.Error(msg, args...) }
