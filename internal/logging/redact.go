package logging

import (
	"strings"
)

// redactWriter wraps the lumberjack file writer and replaces known secret
// strings with [REDACTED] before each write. os.Stdout always receives the
// original unredacted bytes via the MultiWriter in Setup.
type redactWriter struct {
	lgr *Logger
}

func (rw *redactWriter) Write(p []byte) (int, error) {
	rw.lgr.secretsMu.RLock()
	secrets := rw.lgr.secrets
	rw.lgr.secretsMu.RUnlock()

	line := string(p)

	if len(secrets) > 0 {
		pairs := make([]string, 0, len(secrets)*2)
		for _, s := range secrets {
			pairs = append(pairs, s, "[REDACTED]")
		}
		line = strings.NewReplacer(pairs...).Replace(line)
	}

	n, err := rw.lgr.lj.Write([]byte(line))
	if err != nil {
		return n, err
	}
	// Return the original length so slog doesn't treat a shorter redacted
	// line as a short-write error.
	return len(p), nil
}
