package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/getmiroir/trading-service/internal/platform"
	"github.com/getmiroir/trading-service/internal/settings/subsettings"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogType identifies which log file is written to.
type LogType string

const (
	ServiceLog LogType = "service"
	GUILog     LogType = "gui"
)

// Logger wraps a lumberjack rolling-file writer and manages per-write
// secret redaction. It is safe for concurrent use.
type Logger struct {
	lj        *lumberjack.Logger
	secretsMu sync.RWMutex
	secrets   []string
}

// Setup creates a Logger that writes to ~/{cfg.DirName}/{cfg.SubDirName}/{logType}.log in
// addition to os.Stdout. Lumberjack handles rotation, size capping, and
// old-file deletion according to cfg. It replaces the default slog logger.
// The caller must call Close() when done.
func Setup(logType LogType, levelStr string, cfg subsettings.Logging) (*Logger, error) {
	logDir, err := LogDir()
	if err != nil {
		return nil, fmt.Errorf("resolving log dir: %w", err)
	}

	lj := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, string(logType)+".log"),
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
		LocalTime:  cfg.LocalTime,
	}

	lgr := &Logger{lj: lj}
	rw := &redactWriter{lgr: lgr}
	multi := io.MultiWriter(os.Stdout, rw)

	opts := &slog.HandlerOptions{Level: parseLevel(levelStr)}
	slog.SetDefault(slog.New(slog.NewTextHandler(multi, opts)))

	return lgr, nil
}

// SetSecrets registers the set of secret strings that should be redacted from
// log files. Call this after loading settings and again after any account
// change. Empty strings are silently ignored.
func (lgr *Logger) SetSecrets(secrets []string) {
	filtered := make([]string, 0, len(secrets))
	for _, s := range secrets {
		if s != "" {
			filtered = append(filtered, s)
		}
	}
	lgr.secretsMu.Lock()
	lgr.secrets = filtered
	lgr.secretsMu.Unlock()
}

// Close flushes and closes the underlying log file.
func (lgr *Logger) Close() error {
	return lgr.lj.Close()
}

// LogDir returns the logs directory inside the application data directory
// (see platform.DataDir).
func LogDir() (string, error) {
	return platform.LogsDir()
}

func parseLevel(s string) slog.Level {
	if strings.EqualFold(s, "WARNING") {
		s = "WARN"
	}
	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return slog.LevelInfo
	}
	return l
}
