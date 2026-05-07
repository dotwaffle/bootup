package buildinfo

import (
	"strings"
	"testing"
)

func TestCurrentUsesFallbacks(t *testing.T) {
	oldVersion, oldCommit, oldDate, oldDirty := version, commit, date, dirty
	version, commit, date, dirty = "", "", "", ""
	t.Cleanup(func() {
		version, commit, date, dirty = oldVersion, oldCommit, oldDate, oldDirty
	})

	info := Current()
	if info.Version != "devel" {
		t.Fatalf("Version = %q, want devel", info.Version)
	}
	if info.Commit != "unknown" {
		t.Fatalf("Commit = %q, want unknown", info.Commit)
	}
	if info.Date != "unknown" {
		t.Fatalf("Date = %q, want unknown", info.Date)
	}
	if info.Dirty != "unknown" {
		t.Fatalf("Dirty = %q, want unknown", info.Dirty)
	}
	if !strings.HasPrefix(info.GoVersion, "go") {
		t.Fatalf("GoVersion = %q, want Go runtime version", info.GoVersion)
	}
}

func TestCurrentUsesStampedValues(t *testing.T) {
	oldVersion, oldCommit, oldDate, oldDirty := version, commit, date, dirty
	version = "v1.2.3"
	commit = strings.Repeat("a", 40)
	date = "2026-05-07T09:00:00Z"
	dirty = "clean"
	t.Cleanup(func() {
		version, commit, date, dirty = oldVersion, oldCommit, oldDate, oldDirty
	})

	info := Current()
	if info.Version != version || info.Commit != commit || info.Date != date || info.Dirty != dirty {
		t.Fatalf("Current() = %#v, want stamped values", info)
	}
}

func TestFormatText(t *testing.T) {
	info := Info{
		Version:   "v1.2.3",
		Commit:    strings.Repeat("a", 40),
		Date:      "2026-05-07T09:00:00Z",
		Dirty:     "clean",
		GoVersion: "go1.25.0",
	}

	got := FormatText(info)
	want := "bootup version\n" +
		"version\tv1.2.3\n" +
		"commit\t" + strings.Repeat("a", 40) + "\n" +
		"date\t2026-05-07T09:00:00Z\n" +
		"dirty\tclean\n" +
		"go\tgo1.25.0\n"
	if got != want {
		t.Fatalf("FormatText() = %q, want %q", got, want)
	}
}
