// Package observability holds logging concerns for the agenda service. It wraps
// the usecases and the gRPC server from the outside, so neither the domain nor
// the usecases know about logging.
package observability

import (
	"io"
	"log/slog"
	"strings"
)

// NewLogger builds a slog logger. level is one of debug, info, warn or error and
// format is json or text; unknown values fall back to info and json.
func NewLogger(w io.Writer, level, format string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}
	if strings.EqualFold(format, "text") {
		return slog.New(slog.NewTextHandler(w, opts))
	}
	return slog.New(slog.NewJSONHandler(w, opts))
}
