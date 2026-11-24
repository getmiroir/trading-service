package logging

import (
	"log/slog"
	"os"
)

// SetDefaultStdoutLogging sets slog's default handler to a text handler on stdout only.
// Use before settings (and file logging) are available; later call logging.Setup for file output.
// logLevel should be one of: DEBUG, INFO, WARNING, ERROR (case-insensitive).
func SetDefaultStdoutLogging(logLevel string) {
	opts := &slog.HandlerOptions{Level: parseLevel(logLevel)}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, opts)))
}

// NewLogger creates a new slog.Logger with a "service" attribute derived
// from the current default logger.
func NewLogger(service string) *slog.Logger {
	return slog.Default().With("service", service)
}
