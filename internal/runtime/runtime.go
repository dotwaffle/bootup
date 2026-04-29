// Package runtime prepares early-boot host state before provider operations.
package runtime

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"strings"
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

// NetworkPreparer validates networking before provider operations.
type NetworkPreparer struct {
	Interfaces    func() ([]net.Interface, error)
	ReadFile      func(string) ([]byte, error)
	WriteFile     func(string, []byte, fs.FileMode) error
	ResolverPath  string
	KernelPNPPath string
}

// Prepare checks for an already configured non-loopback interface. If the
// kernel was asked to configure networking with ip=, DNS hints from
// /proc/net/pnp are copied into resolv.conf when resolv.conf is absent.
func (p NetworkPreparer) Prepare(ctx context.Context) error {
	_ = ctx

	if err := p.installKernelResolver(); err != nil {
		return err
	}

	interfaces := p.Interfaces
	if interfaces == nil {
		interfaces = net.Interfaces
	}
	links, err := interfaces()
	if err != nil {
		return fmt.Errorf("list network interfaces: %w", err)
	}
	for _, link := range links {
		if link.Flags&net.FlagLoopback == 0 && link.Flags&net.FlagUp != 0 {
			return nil
		}
	}
	return errors.New("no configured non-loopback network interface")
}

func (p NetworkPreparer) installKernelResolver() error {
	resolverPath := p.ResolverPath
	if resolverPath == "" {
		resolverPath = "/etc/resolv.conf"
	}
	kernelPNPPath := p.KernelPNPPath
	if kernelPNPPath == "" {
		kernelPNPPath = "/proc/net/pnp"
	}
	readFile := p.ReadFile
	if readFile == nil {
		readFile = os.ReadFile
	}
	writeFile := p.WriteFile
	if writeFile == nil {
		writeFile = os.WriteFile
	}

	resolver, err := readFile(resolverPath)
	if err == nil && strings.TrimSpace(string(resolver)) != "" {
		return nil
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read resolver config: %w", err)
	}

	kernelPNP, err := readFile(kernelPNPPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read kernel network config: %w", err)
	}
	if !strings.Contains(string(kernelPNP), "nameserver") {
		return nil
	}
	if err := writeFile(resolverPath, kernelPNP, 0o644); err != nil {
		return fmt.Errorf("write resolver config: %w", err)
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
