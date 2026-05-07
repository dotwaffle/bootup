package catalog_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/dotwaffle/bootup/internal/catalog"
	"github.com/dotwaffle/bootup/internal/provider"
)

func TestBuildConformanceReportClassifiesTargets(t *testing.T) {
	t.Parallel()

	linuxTarget := conformanceTarget("opensuse-leap-160-amd64-netboot", "linux", provider.BootActionLinuxKexec)
	linuxTarget.Source = provider.SourceEntry{
		BaseURL:    "https://download.example/opensuse",
		KernelPath: "boot/x86_64/loader/linux",
		InitrdPath: "boot/x86_64/loader/initrd",
	}
	debianTarget := conformanceTarget("debian-trixie-amd64-netboot", "debian", provider.BootActionLinuxKexec)
	ubuntuTarget := conformanceTarget("ubuntu-2604-amd64-netboot", "ubuntu", provider.BootActionLinuxKexec)
	mfsTarget := conformanceTarget("mfsbsd-142-amd64", "mfsbsd", provider.BootActionFreeBSDKboot)
	localTarget := conformanceTarget("local-disk-auto", "local", provider.BootActionLocalBoot)

	registry := provider.NewRegistry()
	for _, stub := range []conformanceProvider{
		{
			id:      "debian",
			targets: []provider.Target{debianTarget},
			plans: map[string]provider.BootPlan{
				debianTarget.ID: {
					Target: debianTarget,
					Kernel: provider.Artifact{URL: "https://mirror.example/linux"},
					Initrd: provider.Artifact{URL: "https://mirror.example/initrd.gz"},
					Verification: provider.Verification{
						MetadataURL: "https://mirror.example/dists/trixie/InRelease",
						ChecksumURL: "https://mirror.example/dists/trixie/images/SHA256SUMS",
					},
				},
			},
		},
		{
			id:      "linux",
			targets: []provider.Target{linuxTarget},
			plans: map[string]provider.BootPlan{
				linuxTarget.ID: {
					Target: linuxTarget,
					Action: provider.BootActionLinuxKexec,
					Kernel: provider.Artifact{URL: "https://download.example/linux"},
					Initrd: provider.Artifact{URL: "https://download.example/initrd"},
				},
			},
		},
		{
			id:      "local",
			targets: []provider.Target{localTarget},
			plans: map[string]provider.BootPlan{
				localTarget.ID: {
					Target: localTarget,
					Action: provider.BootActionLocalBoot,
				},
			},
		},
		{
			id:      "mfsbsd",
			targets: []provider.Target{mfsTarget},
			plans: map[string]provider.BootPlan{
				mfsTarget.ID: {
					Target: mfsTarget,
					Action: provider.BootActionFreeBSDKboot,
					FreeBSDKboot: provider.FreeBSDKbootPlan{
						Payload:       provider.Artifact{URL: "https://mfsbsd.example/mfsbsd.iso", SHA256: strings.Repeat("a", 64)},
						LoaderArchive: provider.Artifact{URL: "https://freebsd.example/base.txz", SHA256: strings.Repeat("b", 64)},
					},
				},
			},
		},
		{
			id:      "ubuntu",
			targets: []provider.Target{ubuntuTarget},
			plans: map[string]provider.BootPlan{
				ubuntuTarget.ID: {
					Target: ubuntuTarget,
					Kernel: provider.Artifact{URL: "https://releases.example/linux"},
					Initrd: provider.Artifact{URL: "https://releases.example/initrd"},
					Verification: provider.Verification{
						ChecksumURL:  "https://releases.example/SHA256SUMS",
						SignatureURL: "https://releases.example/SHA256SUMS.gpg",
					},
				},
			},
		},
	} {
		if err := registry.Register(stub); err != nil {
			t.Fatalf("register %s: %v", stub.id, err)
		}
	}

	report, err := catalog.BuildConformanceReport(context.Background(), registry)
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	if got := report.PlanErrorCount(); got != 0 {
		t.Fatalf("plan error count = %d, want 0", got)
	}

	assertConformanceEntry(t, report, linuxTarget.ID, catalog.PlanStatusOK, catalog.ArtifactTrustHTTPSOnly,
		[]catalog.SmokeCoverage{catalog.SmokeCoverageLiveStage, catalog.SmokeCoverageCatalogQEMU})
	assertConformanceEntry(t, report, debianTarget.ID, catalog.PlanStatusOK, catalog.ArtifactTrustReleaseMetadata,
		[]catalog.SmokeCoverage{catalog.SmokeCoverageDebianQEMU})
	assertConformanceEntry(t, report, ubuntuTarget.ID, catalog.PlanStatusOK, catalog.ArtifactTrustSignedMetadata,
		[]catalog.SmokeCoverage{catalog.SmokeCoverageUbuntuQEMU})
	assertConformanceEntry(t, report, mfsTarget.ID, catalog.PlanStatusOK, catalog.ArtifactTrustHashPinned,
		[]catalog.SmokeCoverage{catalog.SmokeCoverageMFSBSDKbootQEMU})
	assertConformanceEntry(t, report, localTarget.ID, catalog.PlanStatusOK, catalog.ArtifactTrustNotApplicable,
		[]catalog.SmokeCoverage{catalog.SmokeCoverageMetadataOnly})
}

func TestBuildConformanceReportCapturesPlanErrors(t *testing.T) {
	t.Parallel()

	target := conformanceTarget("broken-amd64-netboot", "broken", provider.BootActionLinuxKexec)
	registry := provider.NewRegistry()
	if err := registry.Register(conformanceProvider{
		id:      "broken",
		targets: []provider.Target{target},
		err:     errors.New("provider cannot plan target"),
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	report, err := catalog.BuildConformanceReport(context.Background(), registry)
	if err != nil {
		t.Fatalf("build report: %v", err)
	}

	entry := conformanceEntryByID(t, report, target.ID)
	if entry.PlanStatus != catalog.PlanStatusError {
		t.Fatalf("plan status = %q, want %q", entry.PlanStatus, catalog.PlanStatusError)
	}
	if !strings.Contains(entry.PlanError, "provider cannot plan target") {
		t.Fatalf("plan error = %q, want provider error", entry.PlanError)
	}
	if got := report.PlanErrorCount(); got != 1 {
		t.Fatalf("plan error count = %d, want 1", got)
	}
}

func TestLiveCatalogSmokeSupportedUsesCoverageClassification(t *testing.T) {
	t.Parallel()

	target := conformanceTarget("opensuse-leap-160-amd64-netboot", "linux", provider.BootActionLinuxKexec)
	target.Source = provider.SourceEntry{
		BaseURL:    "https://download.example/opensuse",
		KernelPath: "boot/x86_64/loader/linux",
	}
	if !catalog.LiveCatalogSmokeSupported(target) {
		t.Fatalf("live catalog smoke support = false, want true")
	}

	target = conformanceTarget("mfsbsd-142-amd64", "mfsbsd", provider.BootActionFreeBSDKboot)
	if catalog.LiveCatalogSmokeSupported(target) {
		t.Fatalf("live catalog smoke support = true, want false for dedicated mfsBSD helper")
	}
}

type conformanceProvider struct {
	id      string
	targets []provider.Target
	plans   map[string]provider.BootPlan
	err     error
}

func (p conformanceProvider) ID() string {
	return p.id
}

func (p conformanceProvider) Targets(context.Context) ([]provider.Target, error) {
	return p.targets, nil
}

func (p conformanceProvider) Plan(_ context.Context, input provider.PlanInput) (provider.BootPlan, error) {
	if p.err != nil {
		return provider.BootPlan{}, p.err
	}
	plan, ok := p.plans[input.Target.ID]
	if !ok {
		return provider.BootPlan{}, errors.New("missing plan")
	}
	return plan, nil
}

func conformanceTarget(id string, providerID string, action provider.BootAction) provider.Target {
	return provider.Target{
		ID:         id,
		ProviderID: providerID,
		Name:       id,
		Action:     action,
		Catalog: provider.CatalogEntry{
			Distribution: providerID,
			Release:      "test",
			Architecture: "amd64",
			Kind:         "installer",
		},
	}
}

func assertConformanceEntry(t *testing.T, report catalog.ConformanceReport, id string, status catalog.PlanStatus, trust catalog.ArtifactTrust, smoke []catalog.SmokeCoverage) {
	t.Helper()

	entry := conformanceEntryByID(t, report, id)
	if entry.PlanStatus != status {
		t.Fatalf("%s plan status = %q, want %q", id, entry.PlanStatus, status)
	}
	if entry.ArtifactTrust != trust {
		t.Fatalf("%s artifact trust = %q, want %q", id, entry.ArtifactTrust, trust)
	}
	if !slices.Equal(entry.SmokeCoverage, smoke) {
		t.Fatalf("%s smoke coverage = %q, want %q", id, entry.SmokeCoverage, smoke)
	}
}

func conformanceEntryByID(t *testing.T, report catalog.ConformanceReport, id string) catalog.ConformanceEntry {
	t.Helper()

	for _, entry := range report.Entries {
		if entry.Target.ID == id {
			return entry
		}
	}
	t.Fatalf("report entries = %#v, want %s", report.Entries, id)
	return catalog.ConformanceEntry{}
}
