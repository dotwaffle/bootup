package buildinfo

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	version string
	commit  string
	date    string
	dirty   string
)

// Info describes the build metadata stamped into a bootup binary.
type Info struct {
	Version   string
	Commit    string
	Date      string
	Dirty     string
	GoVersion string
}

// Current returns the build metadata for the running binary.
func Current() Info {
	return Info{
		Version:   fallback(version, "devel"),
		Commit:    fallback(commit, "unknown"),
		Date:      fallback(date, "unknown"),
		Dirty:     fallback(dirty, "unknown"),
		GoVersion: runtime.Version(),
	}
}

// FormatText returns build metadata as tab-separated operator diagnostics.
func FormatText(info Info) string {
	var b strings.Builder
	b.WriteString("bootup version\n")
	fmt.Fprintf(&b, "version\t%s\n", info.Version)
	fmt.Fprintf(&b, "commit\t%s\n", info.Commit)
	fmt.Fprintf(&b, "date\t%s\n", info.Date)
	fmt.Fprintf(&b, "dirty\t%s\n", info.Dirty)
	fmt.Fprintf(&b, "go\t%s\n", info.GoVersion)
	return b.String()
}

func fallback(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
