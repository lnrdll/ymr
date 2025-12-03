package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lmittmann/tint"
)

// Setup configures the global slog.Logger
func Setup(debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	opts := &tint.Options{
		Level:     level,
		AddSource: debug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Remove Timestamp
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}

			// Source Formatting
			if a.Key == slog.SourceKey {
				source, _ := a.Value.Any().(*slog.Source)
				if source != nil {
					filename := filepath.Base(source.File)
					newVal := fmt.Sprintf("%s:%d (%s)", filename, source.Line, source.Function)
					return slog.String(slog.SourceKey, newVal)
				}
			}

			return a
		},
	}

	handler := tint.NewHandler(os.Stderr, opts)

	slog.SetDefault(slog.New(handler))
}
