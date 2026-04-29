// Package logging provides serial-friendly structured logging.
package logging

import (
	"io"
	"log/slog"
)

// NewSerialLogger returns a text logger suitable for serial consoles.
func NewSerialLogger(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	}))
}
