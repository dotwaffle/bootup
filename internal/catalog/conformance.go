package catalog

import (
	"context"
	"errors"
	"net/url"
	"slices"

	"github.com/dotwaffle/bootup/internal/provider"
)

// PlanStatus describes whether a target could be planned during conformance.
type PlanStatus string

const (
	// PlanStatusOK means the provider returned a dry-run boot plan.
	PlanStatusOK PlanStatus = "ok"

	// PlanStatusError means the provider rejected the target during planning.
	PlanStatusError PlanStatus = "error"
)

// ArtifactTrust describes the planned boot-artifact trust posture.
type ArtifactTrust string

const (
	// ArtifactTrustUnknown means trust could not be classified, usually
	// because planning failed.
	ArtifactTrustUnknown ArtifactTrust = "unknown"

	// ArtifactTrustHashPinned means every downloadable artifact has a SHA-256
	// pin in the dry-run plan.
	ArtifactTrustHashPinned ArtifactTrust = "hash-pinned"

	// ArtifactTrustSignedMetadata means the dry-run plan verifies signed
	// metadata before artifact use.
	ArtifactTrustSignedMetadata ArtifactTrust = "signed-metadata"

	// ArtifactTrustReleaseMetadata means the dry-run plan verifies release
	// metadata and checksums before artifact use.
	ArtifactTrustReleaseMetadata ArtifactTrust = "release-metadata"

	// ArtifactTrustHTTPSOnly means artifacts are downloaded over HTTPS without
	// stronger planned hash or signature metadata.
	ArtifactTrustHTTPSOnly ArtifactTrust = "https-only"

	// ArtifactTrustPartialHashes means only some downloadable artifacts have
	// SHA-256 pins.
	ArtifactTrustPartialHashes ArtifactTrust = "partial-hashes"

	// ArtifactTrustNotApplicable means the plan does not download boot
	// artifacts.
	ArtifactTrustNotApplicable ArtifactTrust = "not-applicable"

	// ArtifactTrustUnverified means downloadable artifacts are not hash-pinned,
	// signed-metadata backed, release-metadata backed, or HTTPS-only.
	ArtifactTrustUnverified ArtifactTrust = "unverified"
)

// SmokeCoverage identifies explicit smoke helper coverage for a target.
type SmokeCoverage string

const (
	// SmokeCoverageMetadataOnly means no live or QEMU smoke helper explicitly
	// supports the target.
	SmokeCoverageMetadataOnly SmokeCoverage = "metadata-only"

	// SmokeCoverageLiveStage means BOOTUP_LIVE_CATALOG_SMOKE can stage the
	// target outside a VM.
	SmokeCoverageLiveStage SmokeCoverage = "live-stage"

	// SmokeCoverageCatalogQEMU means scripts/smoke-catalog-target.sh can
	// attempt the target through the catalog QEMU helper.
	SmokeCoverageCatalogQEMU SmokeCoverage = "catalog-qemu"

	// SmokeCoverageDebianQEMU means the dedicated Debian QEMU smoke covers the
	// target.
	SmokeCoverageDebianQEMU SmokeCoverage = "debian-qemu"

	// SmokeCoverageUbuntuQEMU means the dedicated Ubuntu QEMU smoke covers the
	// target.
	SmokeCoverageUbuntuQEMU SmokeCoverage = "ubuntu-qemu"

	// SmokeCoverageMFSBSDKbootQEMU means the dedicated mfsBSD kboot QEMU smoke
	// covers the target.
	SmokeCoverageMFSBSDKbootQEMU SmokeCoverage = "mfsbsd-kboot-qemu"
)

// ConformanceReport describes catalog-wide dry-run conformance results.
type ConformanceReport struct {
	Entries []ConformanceEntry
}

// PlanErrorCount returns the number of targets that failed dry-run planning.
func (r ConformanceReport) PlanErrorCount() int {
	var count int
	for _, entry := range r.Entries {
		if entry.PlanStatus == PlanStatusError {
			count++
		}
	}
	return count
}

// ConformanceEntry describes one target in the catalog conformance matrix.
type ConformanceEntry struct {
	Target        provider.Target
	Action        provider.BootAction
	PlanStatus    PlanStatus
	PlanError     string
	ArtifactTrust ArtifactTrust
	SmokeCoverage []SmokeCoverage
}

// BuildConformanceReport builds a dry-run catalog conformance report.
func BuildConformanceReport(ctx context.Context, registry *provider.Registry) (ConformanceReport, error) {
	if registry == nil {
		return ConformanceReport{}, errors.New("registry is required")
	}
	targets, err := registry.Targets(ctx)
	if err != nil {
		return ConformanceReport{}, err
	}

	report := ConformanceReport{
		Entries: make([]ConformanceEntry, 0, len(targets)),
	}
	for _, target := range targets {
		entry := ConformanceEntry{
			Target:        target,
			Action:        provider.ResolveBootAction(target.Action),
			PlanStatus:    PlanStatusOK,
			ArtifactTrust: ArtifactTrustUnknown,
			SmokeCoverage: SmokeCoverageForTarget(target),
		}
		plan, err := registry.Plan(ctx, provider.PlanInput{Target: target})
		if err != nil {
			entry.PlanStatus = PlanStatusError
			entry.PlanError = err.Error()
			report.Entries = append(report.Entries, entry)
			continue
		}
		entry.Action = plan.ResolvedAction()
		entry.ArtifactTrust = ClassifyArtifactTrust(plan)
		report.Entries = append(report.Entries, entry)
	}
	return report, nil
}

// ClassifyArtifactTrust classifies planned boot artifact trust metadata.
func ClassifyArtifactTrust(plan provider.BootPlan) ArtifactTrust {
	artifacts := downloadableArtifacts(plan)
	if len(artifacts) == 0 {
		return ArtifactTrustNotApplicable
	}

	hasPinned := 0
	for _, artifact := range artifacts {
		if artifact.SHA256 != "" {
			hasPinned++
		}
	}
	switch hasPinned {
	case len(artifacts):
		return ArtifactTrustHashPinned
	case 0:
	default:
		return ArtifactTrustPartialHashes
	}

	if plan.Verification.SignatureURL != "" {
		return ArtifactTrustSignedMetadata
	}
	if plan.Verification.MetadataURL != "" && plan.Verification.ChecksumURL != "" {
		return ArtifactTrustReleaseMetadata
	}
	if artifactsUseHTTPS(artifacts) {
		return ArtifactTrustHTTPSOnly
	}
	return ArtifactTrustUnverified
}

// SmokeCoverageForTarget returns explicit smoke helper coverage for target.
func SmokeCoverageForTarget(target provider.Target) []SmokeCoverage {
	var coverage []SmokeCoverage
	if genericLinuxCatalogSmokeSupported(target) {
		coverage = append(coverage, SmokeCoverageLiveStage, SmokeCoverageCatalogQEMU)
	}
	switch {
	case target.ProviderID == "debian" && target.ID == "debian-trixie-amd64-netboot":
		coverage = append(coverage, SmokeCoverageDebianQEMU)
	case target.ProviderID == "ubuntu" && target.ID == "ubuntu-2604-amd64-netboot":
		coverage = append(coverage, SmokeCoverageUbuntuQEMU)
	case target.ProviderID == "mfsbsd" && target.ID == "mfsbsd-142-amd64" && provider.ResolveBootAction(target.Action) == provider.BootActionFreeBSDKboot:
		coverage = append(coverage, SmokeCoverageMFSBSDKbootQEMU)
	}
	if len(coverage) == 0 {
		return []SmokeCoverage{SmokeCoverageMetadataOnly}
	}
	return coverage
}

// LiveCatalogSmokeSupported reports whether the live catalog staging smoke can
// stage target.
func LiveCatalogSmokeSupported(target provider.Target) bool {
	return slices.Contains(SmokeCoverageForTarget(target), SmokeCoverageLiveStage)
}

func genericLinuxCatalogSmokeSupported(target provider.Target) bool {
	return provider.ResolveBootAction(target.Action) == provider.BootActionLinuxKexec &&
		target.ProviderID == "linux" &&
		target.Source.BaseURL != "" &&
		target.Source.KernelPath != ""
}

func downloadableArtifacts(plan provider.BootPlan) []provider.Artifact {
	switch plan.ResolvedAction() {
	case provider.BootActionFreeBSDKboot:
		return nonEmptyArtifacts(plan.FreeBSDKboot.Payload, plan.FreeBSDKboot.LoaderArchive)
	default:
		return nonEmptyArtifacts(plan.Kernel, plan.Initrd)
	}
}

func nonEmptyArtifacts(artifacts ...provider.Artifact) []provider.Artifact {
	out := make([]provider.Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.URL != "" {
			out = append(out, artifact)
		}
	}
	return out
}

func artifactsUseHTTPS(artifacts []provider.Artifact) bool {
	for _, artifact := range artifacts {
		parsed, err := url.Parse(artifact.URL)
		if err != nil || parsed.Scheme != "https" {
			return false
		}
	}
	return true
}
