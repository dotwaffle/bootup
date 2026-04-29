// Package ui renders operator-facing bootup interfaces.
package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/dotwaffle/bootup/internal/provider"
)

const defaultWidth = 80

// TextMenu renders a serial-friendly text interface.
type TextMenu struct {
	Width int
}

// RenderTargets writes a target list.
func (m TextMenu) RenderTargets(w io.Writer, targets []provider.Target) error {
	if _, err := fmt.Fprintln(w, "bootup targets"); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	for _, target := range targets {
		line := fmt.Sprintf("%s  %s  %s", target.ID, target.Architecture, target.Name)
		if _, err := fmt.Fprintln(w, truncate(line, m.width())); err != nil {
			return fmt.Errorf("write target %s: %w", target.ID, err)
		}
	}
	return nil
}

// RenderProgress writes a progress line.
func (m TextMenu) RenderProgress(w io.Writer, message string) error {
	if _, err := fmt.Fprintln(w, truncate("... "+message, m.width())); err != nil {
		return fmt.Errorf("write progress: %w", err)
	}
	return nil
}

// RenderFatal writes a fatal error line.
func (m TextMenu) RenderFatal(w io.Writer, message string) error {
	if _, err := fmt.Fprintln(w, truncate("fatal: "+message, m.width())); err != nil {
		return fmt.Errorf("write fatal error: %w", err)
	}
	return nil
}

// SelectTargetByID returns the target with id.
func SelectTargetByID(targets []provider.Target, id string) (provider.Target, error) {
	for _, target := range targets {
		if target.ID == id {
			return target, nil
		}
	}
	return provider.Target{}, fmt.Errorf("target %q not found", id)
}

func (m TextMenu) width() int {
	if m.Width <= 0 {
		return defaultWidth
	}
	return m.Width
}

func truncate(s string, width int) string {
	if len(s) <= width {
		return s
	}
	if width <= 1 {
		return s[:width]
	}
	return strings.TrimRight(s[:width-1], " ") + ">"
}
