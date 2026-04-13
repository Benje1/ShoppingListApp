package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// rotatingWriter is an io.Writer that writes to a date-stamped file under logDir.
// It opens a new file when the calendar day changes (UTC), so log files are
// named app-YYYY-MM-DD.log and rotate automatically at midnight.
type rotatingWriter struct {
	mu      sync.Mutex
	logDir  string
	current *os.File
	day     string // the date string of the currently open file
}

func newRotatingWriter(logDir string) (*rotatingWriter, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("creating log directory %q: %w", logDir, err)
	}
	w := &rotatingWriter{logDir: logDir}
	if err := w.rotate(); err != nil {
		return nil, err
	}
	return w, nil
}

// Write implements io.Writer. It rotates the file if the day has changed.
func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := time.Now().UTC().Format("2006-01-02")
	if today != w.day {
		if err := w.rotate(); err != nil {
			// Fall back to stderr so we don't silently lose log lines
			fmt.Fprintf(os.Stderr, "logger: rotate failed: %v\n", err)
		}
	}

	return w.current.Write(p)
}

// rotate opens (or creates) the log file for today, closing the previous one.
// Must be called with w.mu held.
func (w *rotatingWriter) rotate() error {
	today := time.Now().UTC().Format("2006-01-02")
	path := filepath.Join(w.logDir, fmt.Sprintf("app-%s.log", today))

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening log file %q: %w", path, err)
	}

	if w.current != nil {
		_ = w.current.Close()
	}
	w.current = f
	w.day = today
	return nil
}
