package logging_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/dotwaffle/bootup/internal/logging"
)

func TestNewSerialLoggerWritesTextRecords(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	logger := logging.NewSerialLogger(&out, slog.LevelInfo)

	logger.Info("bootup started", slog.String("console", "serial"))

	got := out.String()
	if !strings.Contains(got, "level=INFO") {
		t.Fatalf("log output = %q, want info level", got)
	}
	if !strings.Contains(got, "msg=\"bootup started\"") {
		t.Fatalf("log output = %q, want message", got)
	}
	if !strings.Contains(got, "console=serial") {
		t.Fatalf("log output = %q, want console attr", got)
	}
}
