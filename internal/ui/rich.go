package ui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"io"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dotwaffle/bootup/internal/provider"
)

// ErrSelectionCanceled is returned when the operator exits target selection.
var ErrSelectionCanceled = errors.New("target selection canceled")

// RichMenu renders a bright interactive terminal interface.
type RichMenu struct {
	Width   int
	Stdin   io.Reader
	Stdout  io.Writer
	Animate bool
}

// SelectTarget prompts the operator to choose a target.
func (m RichMenu) SelectTarget(ctx context.Context, targets []provider.Target) (provider.Target, error) {
	if len(targets) == 0 {
		return provider.Target{}, errors.New("no boot targets available")
	}

	picker := NewTargetPicker(targets)
	picker.width = m.width()

	options := []tea.ProgramOption{
		tea.WithContext(ctx),
		tea.WithoutSignalHandler(),
		tea.WithWindowSize(m.width(), 25),
	}
	if m.Stdin != nil {
		options = append(options, tea.WithInput(m.Stdin))
	}
	if m.Stdout != nil {
		options = append(options, tea.WithOutput(m.Stdout))
	}

	finalModel, err := tea.NewProgram(picker, options...).Run()
	if err != nil {
		return provider.Target{}, fmt.Errorf("run rich menu: %w", err)
	}
	finalPicker, ok := finalModel.(TargetPicker)
	if !ok {
		return provider.Target{}, errors.New("rich menu returned unexpected model")
	}
	return finalPicker.Selected()
}

// RenderStatus writes a bright progress line for a boot phase.
func (m RichMenu) RenderStatus(w io.Writer, phase string, message string) error {
	frames := []string{"-", "\\", "|", "/"}
	if !m.Animate {
		return m.renderStatusFrame(w, frames[0], phase, message, false)
	}
	for _, frame := range frames {
		if err := m.renderStatusFrame(w, frame, phase, message, true); err != nil {
			return err
		}
		time.Sleep(40 * time.Millisecond)
	}
	return m.renderStatusFrame(w, frames[0], phase, message, false)
}

// RenderFatal writes a readable rich failure panel.
func (m RichMenu) RenderFatal(w io.Writer, message string) error {
	width := m.width()
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("231")).
		Background(lipgloss.Color("196")).
		Padding(0, 1).
		Render("BOOTUP FAILURE")
	if _, err := fmt.Fprintln(w, header); err != nil {
		return fmt.Errorf("write rich fatal header: %w", err)
	}

	reason := truncate("reason: "+message, width)
	reasonStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231"))
	if _, err := fmt.Fprintln(w, reasonStyle.Render(reason)); err != nil {
		return fmt.Errorf("write rich fatal reason: %w", err)
	}

	hint := truncate("stage-1 environment remains available for diagnostics", width)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	if _, err := fmt.Fprintln(w, hintStyle.Render(hint)); err != nil {
		return fmt.Errorf("write rich fatal hint: %w", err)
	}
	return nil
}

func (m RichMenu) renderStatusFrame(w io.Writer, frame string, phase string, message string, carriageReturn bool) error {
	width := m.width()
	percent := phasePercent(phase)
	barWidth := 24
	if width < 72 {
		barWidth = 14
	}
	bar := progress.New(
		progress.WithWidth(barWidth),
		progress.WithColors(lipgloss.Color("#00D7FF"), lipgloss.Color("#FFAF00")),
		progress.WithFillCharacters('#', '-'),
		progress.WithScaled(true),
	).ViewAs(percent)

	phaseText := lipgloss.NewStyle().
		Bold(true).
		Foreground(phaseColor(phase)).
		Render(strings.ToUpper(phase))
	messageWidth := max(width-barWidth-len(phase)-14, 12)
	line := fmt.Sprintf("%s %s %s %s", frame, phaseText, bar, truncate(message, messageWidth))
	if carriageReturn {
		if _, err := fmt.Fprint(w, "\r"+line); err != nil {
			return fmt.Errorf("write rich status frame: %w", err)
		}
		return nil
	}
	if _, err := fmt.Fprintln(w, line); err != nil {
		return fmt.Errorf("write rich status: %w", err)
	}
	return nil
}

func (m RichMenu) width() int {
	if m.Width <= 0 {
		return defaultWidth
	}
	return m.Width
}

func phasePercent(phase string) float64 {
	switch phase {
	case "planning":
		return 0.25
	case "verifying":
		return 0.50
	case "staging":
		return 0.75
	case "loading":
		return 1.0
	default:
		return 0.10
	}
}

func phaseColor(phase string) color.Color {
	switch phase {
	case "planning":
		return lipgloss.Color("51")
	case "verifying":
		return lipgloss.Color("220")
	case "staging":
		return lipgloss.Color("82")
	case "loading":
		return lipgloss.Color("201")
	default:
		return lipgloss.Color("45")
	}
}

// TargetPicker is the rich interactive boot target picker model.
type TargetPicker struct {
	targets  []provider.Target
	cursor   int
	selected int
	canceled bool
	width    int
	spinner  spinner.Model
}

// NewTargetPicker creates a target picker model.
func NewTargetPicker(targets []provider.Target) TargetPicker {
	s := spinner.New(
		spinner.WithSpinner(spinner.Line),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Bold(true)),
	)
	return TargetPicker{
		targets:  append([]provider.Target(nil), targets...),
		selected: -1,
		width:    defaultWidth,
		spinner:  s,
	}
}

// Init starts picker animation.
func (m TargetPicker) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles keyboard and animation messages.
func (m TargetPicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyPressMsg:
		return m.updateKey(msg)
	}
	return m, nil
}

// View renders the picker.
func (m TargetPicker) View() tea.View {
	return tea.NewView(m.Render())
}

// Render returns the picker content.
func (m TargetPicker) Render() string {
	width := m.viewWidth()
	contentWidth := max(width-2, 20)

	var b strings.Builder
	b.WriteString(m.banner(contentWidth))
	b.WriteString("\n\n")
	previousGroup := ""
	for index, target := range m.targets {
		group := targetGroup(target)
		if group != previousGroup {
			b.WriteString(m.groupLine(group, contentWidth))
			b.WriteString("\n")
			previousGroup = group
		}
		b.WriteString(m.targetLine(index, target, contentWidth))
		b.WriteString("\n")
	}
	if len(m.targets) > 0 {
		b.WriteString("\n")
		b.WriteString(m.detail(contentWidth))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	footer := m.spinner.View() + "  up/down j/k move  enter boot  q quit"
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(truncate(footer, contentWidth)))
	b.WriteString("\n")
	return b.String()
}

// Selected returns the chosen target.
func (m TargetPicker) Selected() (provider.Target, error) {
	if m.canceled {
		return provider.Target{}, ErrSelectionCanceled
	}
	if m.selected < 0 || m.selected >= len(m.targets) {
		return provider.Target{}, errors.New("no target selected")
	}
	return m.targets[m.selected], nil
}

func (m TargetPicker) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc", "q":
		m.canceled = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.targets)-1 {
			m.cursor++
		}
	case "enter":
		m.selected = m.cursor
		return m, tea.Quit
	default:
		index, err := strconv.Atoi(msg.String())
		if err == nil && index >= 1 && index <= len(m.targets) {
			m.cursor = index - 1
			m.selected = m.cursor
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m TargetPicker) banner(width int) string {
	activity := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("220")).
		Render(m.spinner.View())
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("231")).
		Background(lipgloss.Color("201")).
		Padding(0, 1).
		Render("BOOTUP")
	tagline := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("51")).
		Render("dynamic verified netboot")
	line := truncate("select a boot target and hand off with kexec", width)
	return fmt.Sprintf("%s %s  %s\n%s", activity, title, tagline, line)
}

func (m TargetPicker) groupLine(group string, width int) string {
	label := "== " + strings.ToUpper(group) + " =="
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("45")).
		Render(truncate(label, width))
}

func (m TargetPicker) targetLine(index int, target provider.Target, width int) string {
	prefix := fmt.Sprintf("  %2d", index+1)
	nameWidth := max(width-17, 16)
	label := fmt.Sprintf("%s  [READY]  %s", prefix, truncate(target.Name, nameWidth))
	meta := truncate(fmt.Sprintf("%s  %s", catalogLabel(target), target.ID), width-4)
	if index == m.cursor {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("16")).
			Background(lipgloss.Color("51")).
			Padding(0, 1).
			Render("> "+truncate(strings.TrimSpace(label), width-4)) + "\n" +
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("231")).
				Background(lipgloss.Color("24")).
				Padding(0, 1).
				Render("  "+meta)
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Render("  " + truncate(strings.TrimSpace(label), width-4) + "\n    " + meta)
}

func targetGroup(target provider.Target) string {
	parts := make([]string, 0, 2)
	if target.Distribution != "" {
		parts = append(parts, target.Distribution)
	}
	if target.Release != "" {
		parts = append(parts, target.Release)
	}
	if len(parts) == 0 {
		return "targets"
	}
	return strings.Join(parts, " / ")
}

func (m TargetPicker) detail(width int) string {
	target := m.targets[m.cursor]
	detail := fmt.Sprintf("ready: %s | provider: %s | arch: %s", target.ID, target.ProviderID, target.Architecture)
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("220")).
		Render(truncate(detail, width))
}

func (m TargetPicker) viewWidth() int {
	if m.width <= 0 {
		return defaultWidth
	}
	return m.width
}
