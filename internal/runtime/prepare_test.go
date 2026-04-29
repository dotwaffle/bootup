package runtime_test

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/dotwaffle/bootup/internal/runtime"
)

type commandCall struct {
	name string
	args []string
}

type fakeCommandRunner struct {
	calls []commandCall
	err   error
}

func (r *fakeCommandRunner) Run(_ context.Context, name string, args ...string) error {
	r.calls = append(r.calls, commandCall{name: name, args: args})
	return r.err
}

func TestNetworkPreparerAcceptsConfiguredInterface(t *testing.T) {
	t.Parallel()

	preparer := runtime.NetworkPreparer{
		Interfaces: func() ([]net.Interface, error) {
			return []net.Interface{
				{Name: "lo", Flags: net.FlagUp | net.FlagLoopback},
				{Name: "eth0", Flags: net.FlagUp},
			}, nil
		},
		ReadFile: func(string) ([]byte, error) {
			return []byte("nameserver 192.0.2.53\n"), nil
		},
	}

	if err := preparer.Prepare(context.Background()); err != nil {
		t.Fatalf("prepare network: %v", err)
	}
}

func TestNetworkPreparerCopiesKernelResolver(t *testing.T) {
	t.Parallel()

	writes := map[string]string{}
	preparer := runtime.NetworkPreparer{
		Interfaces: func() ([]net.Interface, error) {
			return []net.Interface{{Name: "eth0", Flags: net.FlagUp}}, nil
		},
		ReadFile: func(name string) ([]byte, error) {
			switch name {
			case "/etc/resolv.conf":
				return nil, os.ErrNotExist
			case "/proc/net/pnp":
				return []byte("nameserver 10.0.2.3\n"), nil
			default:
				t.Fatalf("unexpected read path %q", name)
				return nil, nil
			}
		},
		WriteFile: func(name string, data []byte, _ os.FileMode) error {
			writes[name] = string(data)
			return nil
		},
	}

	if err := preparer.Prepare(context.Background()); err != nil {
		t.Fatalf("prepare network: %v", err)
	}
	if writes["/etc/resolv.conf"] != "nameserver 10.0.2.3\n" {
		t.Fatalf("resolver write = %q, want kernel DNS", writes["/etc/resolv.conf"])
	}
}

func TestNetworkPreparerRejectsMissingInterface(t *testing.T) {
	t.Parallel()

	preparer := runtime.NetworkPreparer{
		Interfaces: func() ([]net.Interface, error) {
			return []net.Interface{{Name: "lo", Flags: net.FlagUp | net.FlagLoopback}}, nil
		},
		ReadFile: func(string) ([]byte, error) {
			return nil, os.ErrNotExist
		},
	}

	err := preparer.Prepare(context.Background())
	if err == nil {
		t.Fatal("prepare network succeeded, want missing interface error")
	}
}

func TestCertPreparerRequiresSystemPool(t *testing.T) {
	t.Parallel()

	preparer := runtime.CertPreparer{
		LoadSystemPool: func() (*x509.CertPool, error) {
			return x509.NewCertPool(), nil
		},
	}

	if err := preparer.Prepare(); err != nil {
		t.Fatalf("prepare certs: %v", err)
	}
}

func TestCertPreparerWrapsLoaderFailure(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("missing cert bundle")
	preparer := runtime.CertPreparer{
		LoadSystemPool: func() (*x509.CertPool, error) {
			return nil, wantErr
		},
	}

	err := preparer.Prepare()
	if !errors.Is(err, wantErr) {
		t.Fatalf("prepare error = %v, want wrapped %v", err, wantErr)
	}
}

func TestTimePreparerSkipsSyncWhenClockIsSane(t *testing.T) {
	t.Parallel()

	runner := &fakeCommandRunner{}
	preparer := runtime.TimePreparer{
		Runner:  runner,
		Now:     func() time.Time { return time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC) },
		Minimum: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := preparer.Prepare(context.Background()); err != nil {
		t.Fatalf("prepare time: %v", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("command calls = %#v, want none", runner.calls)
	}
}

func TestTimePreparerSyncsWhenClockIsBeforeMinimum(t *testing.T) {
	t.Parallel()

	runner := &fakeCommandRunner{}
	preparer := runtime.TimePreparer{
		Runner:  runner,
		Now:     func() time.Time { return time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC) },
		Minimum: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := preparer.Prepare(context.Background()); err != nil {
		t.Fatalf("prepare time: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("command calls = %d, want 1", len(runner.calls))
	}
	if runner.calls[0].name != "ntpdate" {
		t.Fatalf("command name = %q, want ntpdate", runner.calls[0].name)
	}
}
