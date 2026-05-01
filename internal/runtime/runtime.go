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

// NetworkConfig describes explicit early-boot network configuration.
type NetworkConfig struct {
	Interface   string
	AddressCIDR string
	Gateway     string
	DNSServers  []string
}

// NetworkPreparer validates networking before provider operations.
type NetworkPreparer struct {
	Interfaces    func() ([]net.Interface, error)
	ReadFile      func(string) ([]byte, error)
	WriteFile     func(string, []byte, fs.FileMode) error
	Runner        CommandRunner
	Config        NetworkConfig
	ResolverPath  string
	KernelPNPPath string
}

// Prepare checks for an already configured non-loopback interface. If the
// kernel was asked to configure networking with ip=, DNS hints from
// /proc/net/pnp are copied into resolv.conf when resolv.conf is absent.
func (p NetworkPreparer) Prepare(ctx context.Context) error {
	if err := p.applyExplicitConfig(ctx); err != nil {
		return err
	}
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

func (p NetworkPreparer) applyExplicitConfig(ctx context.Context) error {
	config := p.Config
	if config.isZero() {
		return nil
	}
	if err := validateNetworkConfig(config); err != nil {
		return err
	}
	if config.Interface != "" {
		runner := p.Runner
		if runner == nil {
			runner = ExecRunner{}
		}
		if err := runner.Run(ctx, "ip", "link", "set", "dev", config.Interface, "up"); err != nil {
			return fmt.Errorf("configure network link: %w", err)
		}
		if config.AddressCIDR != "" {
			if err := runner.Run(ctx, "ip", "addr", "add", config.AddressCIDR, "dev", config.Interface); err != nil {
				return fmt.Errorf("configure network address: %w", err)
			}
		}
		if config.Gateway != "" {
			if err := runner.Run(ctx, "ip", "route", "replace", "default", "via", config.Gateway, "dev", config.Interface); err != nil {
				return fmt.Errorf("configure default route: %w", err)
			}
		}
	}
	if len(config.DNSServers) > 0 {
		if err := p.writeResolver(config.DNSServers); err != nil {
			return err
		}
	}
	return nil
}

func (c NetworkConfig) isZero() bool {
	return c.Interface == "" && c.AddressCIDR == "" && c.Gateway == "" && len(c.DNSServers) == 0
}

func validateNetworkConfig(config NetworkConfig) error {
	if strings.TrimSpace(config.Interface) != config.Interface {
		return errors.New("network interface has surrounding whitespace")
	}
	if (config.AddressCIDR != "" || config.Gateway != "") && config.Interface == "" {
		return errors.New("network interface is required for address or gateway configuration")
	}
	if config.AddressCIDR != "" {
		if _, _, err := net.ParseCIDR(config.AddressCIDR); err != nil {
			return fmt.Errorf("parse network address: %w", err)
		}
	}
	if config.Gateway != "" && net.ParseIP(config.Gateway) == nil {
		return fmt.Errorf("parse network gateway: %q is not an IP address", config.Gateway)
	}
	for _, server := range config.DNSServers {
		if strings.TrimSpace(server) != server {
			return errors.New("DNS server has surrounding whitespace")
		}
		if net.ParseIP(server) == nil {
			return fmt.Errorf("parse DNS server: %q is not an IP address", server)
		}
	}
	return nil
}

func (p NetworkPreparer) writeResolver(servers []string) error {
	resolverPath := p.ResolverPath
	if resolverPath == "" {
		resolverPath = "/etc/resolv.conf"
	}
	writeFile := p.WriteFile
	if writeFile == nil {
		writeFile = os.WriteFile
	}
	var builder strings.Builder
	for _, server := range servers {
		builder.WriteString("nameserver ")
		builder.WriteString(server)
		builder.WriteByte('\n')
	}
	if err := writeFile(resolverPath, []byte(builder.String()), 0o644); err != nil {
		return fmt.Errorf("write resolver config: %w", err)
	}
	return nil
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
