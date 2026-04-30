package ui

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/dotwaffle/bootup/internal/provider"
)

func TestTargetPickerNavigatesAndSelects(t *testing.T) {
	t.Parallel()

	picker := NewTargetPicker(testTargets())
	picker = updatePicker(t, picker, tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	picker = updatePicker(t, picker, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	target, err := picker.Selected()
	if err != nil {
		t.Fatalf("selected target: %v", err)
	}
	if target.ID != "ubuntu-2604-amd64-netboot" {
		t.Fatalf("selected target = %q, want Ubuntu", target.ID)
	}
}

func TestTargetPickerAcceptsNumberSelection(t *testing.T) {
	t.Parallel()

	picker := NewTargetPicker(testTargets())
	picker = updatePicker(t, picker, tea.KeyPressMsg(tea.Key{Code: '2', Text: "2"}))

	target, err := picker.Selected()
	if err != nil {
		t.Fatalf("selected target: %v", err)
	}
	if target.ID != "ubuntu-2604-amd64-netboot" {
		t.Fatalf("selected target = %q, want Ubuntu", target.ID)
	}
}

func TestTargetPickerCancel(t *testing.T) {
	t.Parallel()

	picker := NewTargetPicker(testTargets())
	picker = updatePicker(t, picker, tea.KeyPressMsg(tea.Key{Code: 'q', Text: "q"}))

	if _, err := picker.Selected(); !errors.Is(err, ErrSelectionCanceled) {
		t.Fatalf("selected target error = %v, want cancellation", err)
	}
}

func TestTargetPickerViewRendersMenuContent(t *testing.T) {
	t.Parallel()

	picker := NewTargetPicker(testTargets())
	got := picker.Render()

	for _, want := range []string{
		"BOOTUP",
		"== DEBIAN / TRIXIE ==",
		"== UBUNTU / 26.04 ==",
		"Debian trixie amd64 netboot",
		"Ubuntu 26.04 amd64 netboot",
		"[READY]",
		"enter boot",
		"debian/trixie/amd64/installer",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("view = %q, want %q", got, want)
		}
	}
}

func TestRichMenuRendersStatusAndFatal(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	menu := RichMenu{Width: 80}
	if err := menu.RenderStatus(&out, "planning", "Debian trixie amd64 netboot"); err != nil {
		t.Fatalf("render status: %v", err)
	}
	if err := menu.RenderFatal(&out, "kexec blocked"); err != nil {
		t.Fatalf("render fatal: %v", err)
	}

	got := out.String()
	for _, want := range []string{"PLANNING", "Debian trixie", "BOOTUP FAILURE", "kexec blocked"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output = %q, want %q", got, want)
		}
	}
}

func updatePicker(t *testing.T, picker TargetPicker, msg tea.Msg) TargetPicker {
	t.Helper()

	model, _ := picker.Update(msg)
	updated, ok := model.(TargetPicker)
	if !ok {
		t.Fatalf("updated model = %T, want TargetPicker", model)
	}
	return updated
}

func testTargets() []provider.Target {
	return []provider.Target{
		{
			ID:         "debian-trixie-amd64-netboot",
			ProviderID: "debian",
			Name:       "Debian trixie amd64 netboot",
			Catalog: provider.CatalogEntry{
				Architecture: "amd64",
				Distribution: "debian",
				Release:      "trixie",
				Kind:         "installer",
			},
		},
		{
			ID:         "ubuntu-2604-amd64-netboot",
			ProviderID: "ubuntu",
			Name:       "Ubuntu 26.04 amd64 netboot",
			Catalog: provider.CatalogEntry{
				Architecture: "amd64",
				Distribution: "ubuntu",
				Release:      "26.04",
				Kind:         "installer",
			},
		},
	}
}
