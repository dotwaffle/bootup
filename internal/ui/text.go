// Package ui renders operator-facing bootup interfaces.
package ui

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/dotwaffle/bootup/internal/provider"
)

const defaultWidth = 80

// BootOptionKind identifies the selectable operator entry kind.
type BootOptionKind string

const (
	// BootOptionTarget selects a concrete target.
	BootOptionTarget BootOptionKind = "target"

	// BootOptionDiscoveryFamily selects a family to discover concrete targets.
	BootOptionDiscoveryFamily BootOptionKind = "discovery-family"
)

// BootOption describes one operator-selectable boot menu entry.
type BootOption struct {
	Kind   BootOptionKind
	Target provider.Target
	Family provider.DiscoveryFamily
}

// BootOptions builds a stable combined menu of static targets and discovery
// families.
func BootOptions(targets []provider.Target, families []provider.DiscoveryFamily) []BootOption {
	options := make([]BootOption, 0, len(targets)+len(families))
	for _, target := range targets {
		options = append(options, BootOption{
			Kind:   BootOptionTarget,
			Target: target,
		})
	}
	for _, family := range families {
		options = append(options, BootOption{
			Kind:   BootOptionDiscoveryFamily,
			Family: family,
		})
	}
	return options
}

// TextMenu renders a serial-friendly text interface.
type TextMenu struct {
	Width int
}

// RenderTargets writes a target list.
func (m TextMenu) RenderTargets(w io.Writer, targets []provider.Target) error {
	return m.RenderBootOptions(w, BootOptions(targets, nil))
}

// RenderBootOptions writes a list of static targets and discovery families.
func (m TextMenu) RenderBootOptions(w io.Writer, options []BootOption) error {
	if _, err := fmt.Fprintln(w, "bootup targets"); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	for index, option := range options {
		name := optionName(option)
		if decoration := optionDecoration(option); decoration != "" {
			name += "  " + decoration
		}
		line := fmt.Sprintf("%d  %s  %s  %s", index+1, optionLabel(option), optionID(option), name)
		if _, err := fmt.Fprintln(w, truncate(line, m.width())); err != nil {
			return fmt.Errorf("write boot option %s: %w", optionID(option), err)
		}
	}
	return nil
}

// RenderStatus writes a named phase status line.
func (m TextMenu) RenderStatus(w io.Writer, phase string, message string) error {
	line := fmt.Sprintf("[%s] %s", phase, message)
	if _, err := fmt.Fprintln(w, truncate(line, m.width())); err != nil {
		return fmt.Errorf("write status: %w", err)
	}
	return nil
}

// RenderProgress writes a progress line.
func (m TextMenu) RenderProgress(w io.Writer, message string) error {
	return m.RenderStatus(w, "progress", message)
}

// RenderFatal writes a fatal error line.
func (m TextMenu) RenderFatal(w io.Writer, message string) error {
	if _, err := fmt.Fprintln(w, "bootup failure"); err != nil {
		return fmt.Errorf("write fatal header: %w", err)
	}
	if _, err := fmt.Fprintln(w, truncate("reason: "+message, m.width())); err != nil {
		return fmt.Errorf("write fatal error: %w", err)
	}
	return nil
}

// RenderPrompt writes an input prompt.
func (m TextMenu) RenderPrompt(w io.Writer, prompt string) error {
	if _, err := fmt.Fprint(w, truncate(prompt, m.width())); err != nil {
		return fmt.Errorf("write prompt: %w", err)
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

// SelectTargetByInput returns the target selected by 1-based index or ID.
func SelectTargetByInput(targets []provider.Target, input string) (provider.Target, error) {
	input = strings.TrimSpace(input)
	if index, err := strconv.Atoi(input); err == nil {
		if index >= 1 && index <= len(targets) {
			return targets[index-1], nil
		}
		return provider.Target{}, fmt.Errorf("target index %d out of range", index)
	}
	return SelectTargetByID(targets, input)
}

// SelectBootOptionByInput returns the boot option selected by 1-based index or
// target/family ID.
func SelectBootOptionByInput(options []BootOption, input string) (BootOption, error) {
	input = strings.TrimSpace(input)
	if index, err := strconv.Atoi(input); err == nil {
		if index >= 1 && index <= len(options) {
			return options[index-1], nil
		}
		return BootOption{}, fmt.Errorf("boot option index %d out of range", index)
	}
	for _, option := range options {
		if optionID(option) == input {
			return option, nil
		}
	}
	return BootOption{}, fmt.Errorf("boot option %q not found", input)
}

func (m TextMenu) width() int {
	if m.Width <= 0 {
		return defaultWidth
	}
	return m.Width
}

func optionID(option BootOption) string {
	switch option.Kind {
	case BootOptionTarget:
		return option.Target.ID
	case BootOptionDiscoveryFamily:
		return option.Family.ID
	default:
		return ""
	}
}

func optionName(option BootOption) string {
	switch option.Kind {
	case BootOptionTarget:
		return option.Target.Name
	case BootOptionDiscoveryFamily:
		return option.Family.Name
	default:
		return ""
	}
}

func optionLabel(option BootOption) string {
	switch option.Kind {
	case BootOptionTarget:
		return catalogLabel(option.Target)
	case BootOptionDiscoveryFamily:
		if option.Family.ProviderID != "" {
			return "discovery/" + option.Family.ProviderID
		}
		return "discovery"
	default:
		return "unknown"
	}
}

func optionDecoration(option BootOption) string {
	if option.Kind != BootOptionTarget {
		return ""
	}
	return lifecycleLabel(option.Target.Lifecycle)
}

func catalogLabel(target provider.Target) string {
	parts := make([]string, 0, 4)
	if target.Catalog.Distribution != "" {
		parts = append(parts, target.Catalog.Distribution)
	}
	if target.Catalog.Release != "" {
		parts = append(parts, target.Catalog.Release)
	}
	if target.Catalog.Architecture != "" {
		parts = append(parts, target.Catalog.Architecture)
	}
	if target.Catalog.Kind != "" {
		parts = append(parts, target.Catalog.Kind)
	}
	if len(parts) == 0 {
		return target.Catalog.Architecture
	}
	return strings.Join(parts, "/")
}

func lifecycleLabel(lifecycle provider.LifecycleEntry) string {
	if lifecycle == (provider.LifecycleEntry{}) {
		return ""
	}
	parts := []string{string(lifecycle.Status)}
	if lifecycle.Date != "" {
		parts = append(parts, lifecycle.Date)
	}
	if lifecycle.Source != "" {
		parts = append(parts, lifecycle.Source)
	}
	return "(" + strings.Join(parts, " ") + ")"
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
