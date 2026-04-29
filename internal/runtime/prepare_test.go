package runtime_test

import (
	"context"
	"crypto/x509"
	"errors"
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

func TestNetworkPreparerRunsDHCPClient(t *testing.T) {
	t.Parallel()

	runner := &fakeCommandRunner{}
	preparer := runtime.NetworkPreparer{Runner: runner}

	if err := preparer.Prepare(context.Background()); err != nil {
		t.Fatalf("prepare network: %v", err)
	}

	if len(runner.calls) != 1 {
		t.Fatalf("command calls = %d, want 1", len(runner.calls))
	}
	call := runner.calls[0]
	if call.name != "dhclient" {
		t.Fatalf("command name = %q, want dhclient", call.name)
	}
	if len(call.args) != 2 || call.args[0] != "-ipv4" || call.args[1] != "-ipv6=false" {
		t.Fatalf("command args = %#v, want IPv4-only dhclient args", call.args)
	}
}

func TestNetworkPreparerWrapsCommandFailure(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("link unavailable")
	preparer := runtime.NetworkPreparer{Runner: &fakeCommandRunner{err: wantErr}}

	err := preparer.Prepare(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("prepare error = %v, want wrapped %v", err, wantErr)
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
