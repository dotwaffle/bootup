// Package runtime prepares early-boot host state before provider operations.
package runtime

import (
	"context"
	"crypto/x509"
	"fmt"
	"os/exec"
	"time"
)

// CommandRunner executes a system command.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
}

// ExecRunner runs commands with os/exec.
type ExecRunner struct{}

// Run executes name with args and waits for it to finish.
func (ExecRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %v: %w: %s", name, args, err, output)
	}
	return nil
}

// NetworkPreparer configures networking before provider operations.
type NetworkPreparer struct {
	Runner CommandRunner
}

// Prepare runs the configured DHCP client.
func (p NetworkPreparer) Prepare(ctx context.Context) error {
	runner := p.Runner
	if runner == nil {
		runner = ExecRunner{}
	}
	if err := runner.Run(ctx, "dhclient", "-ipv4", "-ipv6=false"); err != nil {
		return fmt.Errorf("configure dhcp: %w", err)
	}
	return nil
}

// CertPreparer validates that system CA roots are available.
type CertPreparer struct {
	LoadSystemPool func() (*x509.CertPool, error)
}

// Prepare checks that a CA pool can be loaded.
func (p CertPreparer) Prepare() error {
	load := p.LoadSystemPool
	if load == nil {
		load = x509.SystemCertPool
	}
	if _, err := load(); err != nil {
		return fmt.Errorf("load system cert pool: %w", err)
	}
	return nil
}

// TimePreparer ensures system time is sane before TLS-heavy flows.
type TimePreparer struct {
	Runner  CommandRunner
	Now     func() time.Time
	Minimum time.Time
}

// Prepare synchronizes time if the current clock is before the configured
// minimum.
func (p TimePreparer) Prepare(ctx context.Context) error {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	minimum := p.Minimum
	if minimum.IsZero() {
		minimum = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if !now().Before(minimum) {
		return nil
	}

	runner := p.Runner
	if runner == nil {
		runner = ExecRunner{}
	}
	if err := runner.Run(ctx, "ntpdate", "-u", "pool.ntp.org"); err != nil {
		return fmt.Errorf("sync time: %w", err)
	}
	return nil
}
