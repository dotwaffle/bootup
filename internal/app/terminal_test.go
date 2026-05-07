package app

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	expect "github.com/Netflix/go-expect"
)

func TestUseRichMenuAutoUsesPlainForLinuxConsoleTERM(t *testing.T) {
	t.Setenv("TERM", "linux")

	console, err := expect.NewConsole(expect.WithDefaultTimeout(time.Second))
	if err != nil {
		t.Fatalf("create console: %v", err)
	}
	t.Cleanup(func() {
		if err := console.Close(); err != nil {
			t.Fatalf("close console: %v", err)
		}
	})

	runner := New(Config{
		Stdin:  console.Tty(),
		Stdout: console.Tty(),
		Stderr: &bytes.Buffer{},
		Logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		UIMode: UIModeAuto,
	})

	useRich, err := runner.useRichMenu()
	if err != nil {
		t.Fatalf("select UI mode: %v", err)
	}
	if useRich {
		t.Fatal("auto UI selected rich mode for TERM=linux, want plain mode")
	}
}

func TestUseRichMenuForcedRichIgnoresLinuxConsoleTERM(t *testing.T) {
	t.Setenv("TERM", "linux")

	console, err := expect.NewConsole(expect.WithDefaultTimeout(time.Second))
	if err != nil {
		t.Fatalf("create console: %v", err)
	}
	t.Cleanup(func() {
		if err := console.Close(); err != nil {
			t.Fatalf("close console: %v", err)
		}
	})

	runner := New(Config{
		Stdin:  console.Tty(),
		Stdout: console.Tty(),
		Stderr: &bytes.Buffer{},
		Logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		UIMode: UIModeRich,
	})

	useRich, err := runner.useRichMenu()
	if err != nil {
		t.Fatalf("select UI mode: %v", err)
	}
	if !useRich {
		t.Fatal("forced rich UI selected plain mode")
	}
}
