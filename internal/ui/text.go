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
		if option.Kind == BootOptionTarget {
			if err := m.renderTargetOption(w, index+1, option.Target); err != nil {
				return err
			}
			continue
		}
		line := fmt.Sprintf("%d  %s  %s  %s", index+1, optionLabel(option), optionID(option), optionName(option))
		if err := m.writeLine(w, line); err != nil {
			return fmt.Errorf("write boot option %s: %w", optionID(option), err)
		}
	}
	return nil
}

// RenderTargetDetails writes detailed metadata for one target.
func (m TextMenu) RenderTargetDetails(w io.Writer, target provider.Target) error {
	if err := m.writeLine(w, "bootup target"); err != nil {
		return fmt.Errorf("write target header: %w", err)
	}
	lines := []string{
		"id: " + target.ID,
		"name: " + target.Name,
		"provider: " + target.ProviderID,
		"action: " + string(provider.ResolveBootAction(target.Action)),
		"distribution: " + target.Catalog.Distribution,
		"release: " + target.Catalog.Release,
		"architecture: " + target.Catalog.Architecture,
		"kind: " + target.Catalog.Kind,
	}
	for _, line := range lines {
		if err := m.writeLine(w, line); err != nil {
			return fmt.Errorf("write target detail: %w", err)
		}
	}
	if lifecycle := targetLifecycleDetail(target.Lifecycle); lifecycle != "" {
		if err := m.writeLine(w, "lifecycle: "+lifecycle); err != nil {
			return fmt.Errorf("write target lifecycle: %w", err)
		}
	}
	if err := m.renderSourceDetails(w, target.Source); err != nil {
		return err
	}
	if err := m.renderOptionDetails(w, target.Options); err != nil {
		return err
	}
	if err := m.renderSecretDetails(w, target.Secrets); err != nil {
		return err
	}
	return nil
}

// RenderStatus writes a named phase status line.
func (m TextMenu) RenderStatus(w io.Writer, phase string, message string) error {
	line := fmt.Sprintf("[%s] %s", phase, message)
	if err := m.writeLine(w, line); err != nil {
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
	if err := m.writeLine(w, "reason: "+message); err != nil {
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

func (m TextMenu) renderTargetOption(w io.Writer, index int, target provider.Target) error {
	name := target.Name
	if decoration := lifecycleLabel(target.Lifecycle); decoration != "" {
		name += "  " + decoration
	}
	line := fmt.Sprintf("%d  %s  %s  %s", index, catalogLabel(target), target.ID, name)
	if err := m.writeLine(w, line); err != nil {
		return fmt.Errorf("write boot option %s: %w", target.ID, err)
	}
	metadata := fmt.Sprintf("   distribution=%s release=%s architecture=%s kind=%s provider=%s action=%s",
		target.Catalog.Distribution,
		target.Catalog.Release,
		target.Catalog.Architecture,
		target.Catalog.Kind,
		target.ProviderID,
		provider.ResolveBootAction(target.Action),
	)
	if err := m.writeLine(w, metadata); err != nil {
		return fmt.Errorf("write boot option metadata %s: %w", target.ID, err)
	}
	return nil
}

func (m TextMenu) renderSourceDetails(w io.Writer, source provider.SourceEntry) error {
	if source == (provider.SourceEntry{}) {
		return nil
	}
	if err := m.writeLine(w, "source:"); err != nil {
		return fmt.Errorf("write source header: %w", err)
	}
	for _, line := range []string{
		"  base_url: " + source.BaseURL,
		"  iso_name: " + source.ISOName,
		"  iso_sha256: " + source.ISOSHA256,
		"  kernel_path: " + source.KernelPath,
		"  initrd_path: " + source.InitrdPath,
		"  cmdline: " + source.Cmdline,
	} {
		if strings.HasSuffix(line, ": ") {
			continue
		}
		if err := m.writeLine(w, line); err != nil {
			return fmt.Errorf("write source detail: %w", err)
		}
	}
	return nil
}

func (m TextMenu) renderOptionDetails(w io.Writer, options []provider.TargetOption) error {
	if len(options) == 0 {
		return nil
	}
	if err := m.writeLine(w, "options:"); err != nil {
		return fmt.Errorf("write options header: %w", err)
	}
	for _, option := range options {
		line := fmt.Sprintf("  - %s %s %s", option.ID, option.Type, option.Label)
		switch option.Type {
		case provider.TargetOptionBool:
			line += " fragment=" + option.Fragment
		case provider.TargetOptionString:
			line += " template=" + option.Template
		case provider.TargetOptionEnum:
			line += " values=" + optionValuesLabel(option.Values)
		}
		if err := m.writeLine(w, line); err != nil {
			return fmt.Errorf("write option detail %s: %w", option.ID, err)
		}
	}
	return nil
}

func (m TextMenu) renderSecretDetails(w io.Writer, secrets []provider.SecretInput) error {
	if len(secrets) == 0 {
		return nil
	}
	if err := m.writeLine(w, "secrets:"); err != nil {
		return fmt.Errorf("write secrets header: %w", err)
	}
	for _, secret := range secrets {
		requirement := "optional"
		if secret.Required {
			requirement = "required"
		}
		line := fmt.Sprintf("  - %s %s %s %s purpose=%s", secret.ID, requirement, secret.Delivery, secret.Label, secret.Purpose)
		if err := m.writeLine(w, line); err != nil {
			return fmt.Errorf("write secret detail %s: %w", secret.ID, err)
		}
	}
	return nil
}

func (m TextMenu) writeLine(w io.Writer, line string) error {
	_, err := fmt.Fprintln(w, truncate(line, m.width()))
	return err
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

func targetLifecycleDetail(lifecycle provider.LifecycleEntry) string {
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
	return strings.Join(parts, " ")
}

func optionValuesLabel(values []provider.TargetOptionValue) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		part := value.Value
		if value.Fragment != "" {
			part += ":" + value.Fragment
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ",")
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
