package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/dotwaffle/bootup/internal/app"
	"github.com/dotwaffle/bootup/internal/buildinfo"
	"github.com/dotwaffle/bootup/internal/catalog"
	"github.com/dotwaffle/bootup/internal/diagnostics"
	"github.com/dotwaffle/bootup/internal/handoff"
	"github.com/dotwaffle/bootup/internal/logging"
	"github.com/dotwaffle/bootup/internal/policy"
	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providerconfig"
	"github.com/dotwaffle/bootup/internal/runtime"
	bootsecrets "github.com/dotwaffle/bootup/internal/secrets"

	_ "github.com/breml/rootcerts"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "bootup: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	return runWithIO(ctx, args, os.Stdin, os.Stdout, os.Stderr)
}

type namedPreparer struct {
	name    string
	prepare func(context.Context) error
}

func (p namedPreparer) Name() string {
	return p.name
}

func (p namedPreparer) Prepare(ctx context.Context) error {
	return p.prepare(ctx)
}

func runWithIO(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	flags := flag.NewFlagSet("bootup", flag.ContinueOnError)
	flags.SetOutput(stderr)

	showVersion := flags.Bool("version", false, "print build version and exit")
	mode := flags.String("mode", string(app.ModeListTargets), "startup mode")
	uiMode := flags.String("ui", string(app.UIModeAuto), "menu UI mode: auto, rich, plain")
	targetID := flags.String("target", "", "target ID for non-interactive modes")
	discoveryFamilyID := flags.String("discovery-family", "", "discovery family ID for discover-targets mode")
	stagingDir := flags.String("staging-dir", "/tmp/bootup", "directory for verified boot artifacts")
	catalogPath := flags.String("catalog", "", "static provider catalog JSON path")
	catalogURL := flags.String("catalog-url", "", "hosted static provider catalog URL")
	catalogSHA256 := flags.String("catalog-sha256", "", "SHA-256 hex digest for hosted catalog bytes")
	catalogSignaturePath := flags.String("catalog-signature", "", "detached Ed25519 signature path for hosted catalog bytes")
	catalogPublicKeyPath := flags.String("catalog-public-key", "", "Ed25519 public key path for hosted catalog signature")
	catalogMaxAge := flags.Duration("catalog-max-age", 0, "maximum age for hosted catalog published_at metadata")
	catalogRequireFreshness := flags.Bool("catalog-require-freshness", false, "require hosted catalog published_at or expires_at metadata")
	catalogCachePath := flags.String("catalog-cache", "", "hosted catalog cache file path")
	catalogCacheFallback := flags.Bool("catalog-cache-fallback", false, "fall back to authenticated hosted catalog cache on fetch failure")
	catalogMaxBytes := flags.Int64("catalog-max-bytes", 0, "maximum hosted catalog size in bytes")
	catalogIncludeDefault := flags.Bool("catalog-include-default", false, "include embedded default catalog with selected catalog")
	policyFilePath := flags.String("policy-file", "", "signed local dynamic policy decision JSON path")
	policyURL := flags.String("policy-url", "", "signed remote dynamic policy decision URL")
	policySignaturePath := flags.String("policy-signature", "", "detached Ed25519 signature path for policy decision bytes")
	policyPublicKeyPath := flags.String("policy-public-key", "", "Ed25519 public key path for policy decision signature")
	policyMaxAge := flags.Duration("policy-max-age", 0, "maximum age for policy published_at metadata")
	policyTimeout := flags.Duration("policy-timeout", 30*time.Second, "remote policy request timeout")
	policyCachePath := flags.String("policy-cache", "", "dynamic policy cache file path")
	policyCacheFallback := flags.Bool("policy-cache-fallback", false, "fall back to authenticated policy cache on source read failure")
	policyMaxBytes := flags.Int64("policy-max-bytes", 0, "maximum policy decision size in bytes")
	policyFallback := flags.String("policy-fallback", string(policyFallbackNone), "policy failure fallback: none, manual")
	providerConfigPath := flags.String("provider-config", "", "provider runtime config JSON path")
	diagnosticsDir := flags.String("diagnostics-dir", "", "directory for opt-in failure diagnostics bundles")
	consoleMirror := flags.String("console-mirror", "", "also write stdout and stderr to this console path, for example /dev/tty0")
	cmdlineAppend := flags.String("append-cmdline", "", "additional kernel command-line parameters for selected targets")
	var targetOptions optionFlags
	flags.Var(&targetOptions, "option", "target option selection as id=value; repeatable")
	var secretInputs secretFlags
	flags.Var(&secretInputs, "secret", "secret input as id=/absolute/path; repeatable")
	netIface := flags.String("net-iface", "", "network interface to configure before provider operations")
	netAddress := flags.String("net-address", "", "CIDR address to configure before provider operations")
	netGateway := flags.String("net-gateway", "", "default gateway to configure before provider operations")
	netDNS := flags.String("net-dns", "", "comma-separated DNS servers to configure before provider operations")
	hold := flags.Bool("hold", false, "wait after the selected mode completes")
	prepareRuntime := flags.Bool("prepare-runtime", false, "validate network, CA roots, and time before provider operations")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if *showVersion {
		_, err := io.WriteString(stdout, buildinfo.FormatText(buildinfo.Current()))
		return err
	}

	catalogSource := catalogSource{
		Path:              *catalogPath,
		URL:               *catalogURL,
		SHA256:            *catalogSHA256,
		SignaturePath:     *catalogSignaturePath,
		PublicKeyPath:     *catalogPublicKeyPath,
		MaxAge:            *catalogMaxAge,
		RequireFreshness:  *catalogRequireFreshness,
		CachePath:         *catalogCachePath,
		CacheFallback:     *catalogCacheFallback,
		MaxBytes:          *catalogMaxBytes,
		IncludeDefault:    *catalogIncludeDefault,
		CompiledProviders: compiledProviderIDs(),
	}
	policySource := policySource{
		Path:          *policyFilePath,
		URL:           *policyURL,
		SignaturePath: *policySignaturePath,
		PublicKeyPath: *policyPublicKeyPath,
		MaxAge:        *policyMaxAge,
		Timeout:       *policyTimeout,
		CachePath:     *policyCachePath,
		CacheFallback: *policyCacheFallback,
		MaxBytes:      *policyMaxBytes,
		Fallback:      policyFallbackMode(*policyFallback),
	}
	var capture *diagnostics.Capture
	if strings.TrimSpace(*diagnosticsDir) != "" {
		capture = diagnostics.NewCapture()
		stdout = capture.Stdout(stdout)
		stderr = capture.Stderr(stderr)
	}
	if strings.TrimSpace(*consoleMirror) != "" {
		mirror, err := os.OpenFile(*consoleMirror, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			if _, writeErr := fmt.Fprintf(stderr, "bootup: console mirror unavailable path=%s error=%v\n", *consoleMirror, err); writeErr != nil {
				return fmt.Errorf("write console mirror warning: %w", writeErr)
			}
		} else if outputAlreadyIncludes(mirror, stdout, stderr) {
			if closeErr := mirror.Close(); closeErr != nil {
				return fmt.Errorf("close skipped console mirror: %w", closeErr)
			}
		} else {
			defer func() { _ = mirror.Close() }()
			stdout = io.MultiWriter(stdout, mirror)
			stderr = io.MultiWriter(stderr, mirror)
			if _, writeErr := fmt.Fprintf(stderr, "bootup: console mirror enabled path=%s\n", *consoleMirror); writeErr != nil {
				return fmt.Errorf("write console mirror notice: %w", writeErr)
			}
		}
	}

	effectiveMode := app.Mode(*mode)
	effectiveTargetID := *targetID
	effectiveTargetOptions := []provider.SelectedOption(targetOptions)
	var effectiveSecretRefs []provider.SecretRef
	policyPosture := diagnosticsPolicyPosture(policySource)

	runErr := func() error {
		if err := validatePolicyFallback(policySource.Fallback); err != nil {
			return err
		}

		var providerConfig providerconfig.Config
		if *providerConfigPath != "" {
			config, err := providerconfig.LoadFile(*providerConfigPath)
			if err != nil {
				return fmt.Errorf("load provider config: %w", err)
			}
			providerConfig = config
		}

		catalogDoc, err := loadCatalog(ctx, catalogSource)
		if err != nil {
			return fmt.Errorf("load catalog: %w", err)
		}

		var preparers []app.Preparer
		networkConfig := runtime.NetworkConfig{
			Interface:   *netIface,
			AddressCIDR: *netAddress,
			Gateway:     *netGateway,
			DNSServers:  parseDNSServers(*netDNS),
		}
		if *prepareRuntime || hasNetworkConfig(networkConfig) {
			preparers = append(preparers, namedPreparer{
				name: "network",
				prepare: func(ctx context.Context) error {
					return runtime.NetworkPreparer{Config: networkConfig}.Prepare(ctx)
				},
			})
		}
		if *prepareRuntime {
			preparers = append(preparers,
				namedPreparer{
					name: "certificates",
					prepare: func(context.Context) error {
						return runtime.CertPreparer{}.Prepare()
					},
				},
				namedPreparer{
					name: "time",
					prepare: func(ctx context.Context) error {
						return runtime.TimePreparer{}.Prepare(ctx)
					},
				},
			)
		}

		registry := provider.NewRegistry()
		if err := registerProviders(registry, providerConfig, catalogDoc); err != nil {
			return err
		}
		secretStore, err := bootsecrets.Load([]bootsecrets.Selection(secretInputs), bootsecrets.Options{})
		if err != nil {
			return fmt.Errorf("load secret inputs: %w", err)
		}
		if policySource.configured() || effectiveMode == modePolicyTarget {
			selection, decision, selectedMode, err := resolvePolicySelection(ctx, registry, policySource, policySelectionInput{
				Mode:              effectiveMode,
				TargetID:          effectiveTargetID,
				TargetOptions:     effectiveTargetOptions,
				DiscoveryFamilyID: *discoveryFamilyID,
				Secrets:           secretStore,
			})
			if err != nil {
				if policySource.manualFallback(effectiveMode, err) {
					policyPosture = policyPostureWithFallback(policyPosture, string(policyFallbackManual))
					if _, writeErr := fmt.Fprintln(stdout, "policy failure; falling back to manual target selection"); writeErr != nil {
						return fmt.Errorf("write policy fallback notice: %w", writeErr)
					}
				} else {
					return fmt.Errorf("resolve policy: %w", err)
				}
			} else {
				effectiveMode = selectedMode
				effectiveTargetID = selection.Target.ID
				effectiveTargetOptions = selection.Options
				effectiveSecretRefs = selection.SecretRefs
				policyPosture = policyPostureWithDecision(policyPosture, decision)
			}
		}

		runner := app.New(app.Config{
			Registry:          registry,
			Stdin:             stdin,
			Stdout:            stdout,
			Stderr:            stderr,
			Logger:            logging.NewSerialLogger(stderr, slog.LevelInfo),
			Mode:              effectiveMode,
			UIMode:            app.UIMode(*uiMode),
			TargetID:          effectiveTargetID,
			TargetOptions:     effectiveTargetOptions,
			SecretStore:       secretStore,
			SecretRefs:        effectiveSecretRefs,
			DiscoveryFamilyID: *discoveryFamilyID,
			StagingDir:        *stagingDir,
			CmdlineAppend:     *cmdlineAppend,
			Hold:              *hold,
			Executor:          handoff.KexecExecutor{},
			Preparers:         preparers,
		})
		return runner.Run(ctx)
	}()
	if runErr != nil && capture != nil {
		return writeFailureDiagnostics(runErr, failureDiagnosticsInput{
			RootDir:           *diagnosticsDir,
			Capture:           capture,
			Mode:              *mode,
			TargetID:          effectiveTargetID,
			DiscoveryFamilyID: *discoveryFamilyID,
			TargetOptions:     effectiveTargetOptions,
			SecretInputIDs:    secretInputIDs([]bootsecrets.Selection(secretInputs)),
			SecretRefIDs:      secretRefIDs(effectiveSecretRefs),
			SecretRedactions:  secretRedactValues([]bootsecrets.Selection(secretInputs)),
			Policy:            policyPosture,
			PolicyRedactions:  diagnosticsPolicyRedactValues(policySource),
			CatalogSource:     catalogSource,
			ProviderConfig:    *providerConfigPath,
		})
	}
	return runErr
}

func outputAlreadyIncludes(file *os.File, writers ...io.Writer) bool {
	fileInfo, err := file.Stat()
	if err != nil {
		return false
	}
	for _, writer := range writers {
		output, ok := writer.(*os.File)
		if !ok {
			continue
		}
		outputInfo, err := output.Stat()
		if err != nil {
			continue
		}
		if os.SameFile(fileInfo, outputInfo) {
			return true
		}
	}
	return false
}

type failureDiagnosticsInput struct {
	RootDir           string
	Capture           *diagnostics.Capture
	Mode              string
	TargetID          string
	DiscoveryFamilyID string
	TargetOptions     []provider.SelectedOption
	SecretInputIDs    []string
	SecretRefIDs      []string
	SecretRedactions  []string
	Policy            diagnostics.PolicyPosture
	PolicyRedactions  []string
	CatalogSource     catalogSource
	ProviderConfig    string
}

func writeFailureDiagnostics(runErr error, input failureDiagnosticsInput) error {
	summary := diagnostics.BuildSummary(diagnostics.SummaryInput{
		Mode:              input.Mode,
		TargetID:          input.TargetID,
		DiscoveryFamilyID: input.DiscoveryFamilyID,
		SelectedOptions:   input.TargetOptions,
		SecretInputIDs:    input.SecretInputIDs,
		SecretRefIDs:      input.SecretRefIDs,
		Catalog:           diagnosticsCatalogPosture(input.CatalogSource),
		ProviderConfig:    diagnostics.ProviderConfigPosture{PathSet: input.ProviderConfig != ""},
		Policy:            input.Policy,
		Error:             runErr,
		RedactValues:      append(append(diagnosticsRedactValues(input.CatalogSource, input.ProviderConfig), input.SecretRedactions...), input.PolicyRedactions...),
	})
	bundleDir, diagnosticsErr := diagnostics.WriteBundle(diagnostics.Bundle{
		RootDir: input.RootDir,
		Summary: summary,
		Stdout:  input.Capture.StdoutBytes(),
		Stderr:  input.Capture.StderrBytes(),
	})
	if diagnosticsErr != nil {
		return fmt.Errorf("%w; write diagnostics: %w", runErr, diagnosticsErr)
	}
	return fmt.Errorf("%w; diagnostics: %s", runErr, bundleDir)
}

func diagnosticsCatalogPosture(source catalogSource) diagnostics.CatalogPosture {
	posture := diagnostics.CatalogPosture{
		LocalPathSet:   source.Path != "",
		HostedURLSet:   source.URL != "",
		SHA256:         strings.TrimSpace(source.SHA256) != "",
		Ed25519:        source.SignaturePath != "" || source.PublicKeyPath != "",
		SignatureFiles: source.SignaturePath != "" || source.PublicKeyPath != "",
		Freshness:      source.RequireFreshness || source.MaxAge > 0,
		Cache:          source.CachePath != "",
		CacheFallback:  source.CacheFallback,
		MaxBytes:       source.MaxBytes > 0,
	}
	switch {
	case source.Path != "" && source.URL != "":
		posture.Source = "multiple"
	case source.URL != "":
		posture.Source = "hosted"
	case source.Path != "":
		posture.Source = "local"
	default:
		posture.Source = "embedded"
	}
	return posture
}

func diagnosticsRedactValues(source catalogSource, providerConfigPath string) []string {
	values := []string{
		source.Path,
		source.URL,
		source.SignaturePath,
		source.PublicKeyPath,
		source.CachePath,
		providerConfigPath,
	}
	redactions := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			redactions = append(redactions, value)
		}
	}
	return redactions
}

func secretInputIDs(selections []bootsecrets.Selection) []string {
	ids := make([]string, 0, len(selections))
	for _, selection := range selections {
		id := strings.TrimSpace(selection.ID)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func secretRedactValues(selections []bootsecrets.Selection) []string {
	values := make([]string, 0, len(selections))
	for _, selection := range selections {
		path := strings.TrimSpace(selection.Path)
		if path != "" {
			values = append(values, path)
		}
	}
	return values
}

func secretRefIDs(refs []provider.SecretRef) []string {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.ID != "" {
			ids = append(ids, ref.ID)
		}
	}
	return ids
}

type optionFlags []provider.SelectedOption

func (f *optionFlags) String() string {
	if f == nil || len(*f) == 0 {
		return ""
	}
	parts := make([]string, 0, len(*f))
	for _, option := range *f {
		parts = append(parts, option.ID+"="+option.Value)
	}
	return strings.Join(parts, ",")
}

func (f *optionFlags) Set(value string) error {
	id, optionValue, ok := strings.Cut(value, "=")
	if !ok {
		return errors.New("target option must use id=value")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("target option ID is required")
	}
	if optionValue == "" {
		return fmt.Errorf("target option %s value is required", id)
	}
	*f = append(*f, provider.SelectedOption{ID: id, Value: optionValue})
	return nil
}

type secretFlags []bootsecrets.Selection

func (f *secretFlags) String() string {
	if f == nil || len(*f) == 0 {
		return ""
	}
	parts := make([]string, 0, len(*f))
	for _, selection := range *f {
		parts = append(parts, selection.ID+"=<redacted>")
	}
	return strings.Join(parts, ",")
}

func (f *secretFlags) Set(value string) error {
	id, path, ok := strings.Cut(value, "=")
	if !ok {
		return errors.New("secret input must use id=/absolute/path")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("secret input ID is required")
	}
	if path == "" {
		return fmt.Errorf("secret input %s path is required", id)
	}
	*f = append(*f, bootsecrets.Selection{ID: id, Path: path})
	return nil
}

func parseDNSServers(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	servers := make([]string, 0, len(parts))
	for _, part := range parts {
		if server := strings.TrimSpace(part); server != "" {
			servers = append(servers, server)
		}
	}
	return servers
}

func hasNetworkConfig(config runtime.NetworkConfig) bool {
	return config.Interface != "" || config.AddressCIDR != "" || config.Gateway != "" || len(config.DNSServers) > 0
}

const modePolicyTarget app.Mode = "policy-target"

type policyFallbackMode string

const (
	policyFallbackNone   policyFallbackMode = "none"
	policyFallbackManual policyFallbackMode = "manual"
)

type policySource struct {
	Path          string
	URL           string
	SignaturePath string
	PublicKeyPath string
	MaxAge        time.Duration
	Timeout       time.Duration
	CachePath     string
	CacheFallback bool
	MaxBytes      int64
	Fallback      policyFallbackMode
}

func (s policySource) configured() bool {
	return s.Path != "" ||
		s.URL != "" ||
		s.SignaturePath != "" ||
		s.PublicKeyPath != "" ||
		s.MaxAge > 0 ||
		s.CachePath != "" ||
		s.CacheFallback ||
		s.MaxBytes > 0
}

func (s policySource) manualFallback(mode app.Mode, err error) bool {
	return mode == app.ModeMenu && s.Fallback == policyFallbackManual && errors.Is(err, policy.ErrInvalidPolicy)
}

func validatePolicyFallback(fallback policyFallbackMode) error {
	if fallback == "" || fallback == policyFallbackNone || fallback == policyFallbackManual {
		return nil
	}
	return fmt.Errorf("%w: unsupported policy fallback %q", policy.ErrInvalidPolicy, fallback)
}

type policySelectionInput struct {
	Mode              app.Mode
	TargetID          string
	TargetOptions     []provider.SelectedOption
	DiscoveryFamilyID string
	Secrets           provider.SecretStore
}

func resolvePolicySelection(ctx context.Context, registry *provider.Registry, source policySource, input policySelectionInput) (policy.Selection, policy.Decision, app.Mode, error) {
	selectedMode, err := policyAppMode(input.Mode, source)
	if err != nil {
		return policy.Selection{}, policy.Decision{}, "", err
	}
	if source.Path != "" && source.URL != "" {
		return policy.Selection{}, policy.Decision{}, "", fmt.Errorf("%w: cannot use --policy-file with --policy-url", policy.ErrInvalidPolicy)
	}
	if source.Fallback == policyFallbackManual && input.Mode != app.ModeMenu {
		return policy.Selection{}, policy.Decision{}, "", fmt.Errorf("%w: --policy-fallback=manual requires --mode=menu", policy.ErrInvalidPolicy)
	}
	if input.TargetID != "" {
		return policy.Selection{}, policy.Decision{}, "", fmt.Errorf("%w: cannot combine --target with dynamic policy", policy.ErrInvalidPolicy)
	}
	if len(input.TargetOptions) != 0 {
		return policy.Selection{}, policy.Decision{}, "", fmt.Errorf("%w: cannot combine --option with dynamic policy", policy.ErrInvalidPolicy)
	}
	if input.DiscoveryFamilyID != "" {
		return policy.Selection{}, policy.Decision{}, "", fmt.Errorf("%w: cannot combine --discovery-family with dynamic policy", policy.ErrInvalidPolicy)
	}
	trust, err := policyTrust(source)
	if err != nil {
		return policy.Selection{}, policy.Decision{}, "", err
	}
	decision, err := loadPolicyDecision(ctx, source, trust)
	if err != nil {
		return policy.Selection{}, policy.Decision{}, "", err
	}
	targets, err := registry.Targets(ctx)
	if err != nil {
		return policy.Selection{}, policy.Decision{}, "", fmt.Errorf("%w: list target inventory: %w", policy.ErrInvalidPolicy, err)
	}
	selection, err := policy.Validate(policy.ValidateInput{
		Decision: decision,
		Targets:  targets,
		Secrets:  input.Secrets,
	})
	if err != nil {
		return policy.Selection{}, policy.Decision{}, "", err
	}
	return selection, decision, selectedMode, nil
}

func loadPolicyDecision(ctx context.Context, source policySource, trust policy.Trust) (policy.Decision, error) {
	if source.URL != "" {
		return policy.LoadURL(ctx, policy.RemoteOptions{
			URL:           source.URL,
			Trust:         trust,
			MaxAge:        source.MaxAge,
			Timeout:       source.Timeout,
			CachePath:     source.CachePath,
			CacheFallback: source.CacheFallback,
			MaxBytes:      source.MaxBytes,
		})
	}
	return policy.LoadFile(policy.LoadOptions{
		Path:          source.Path,
		Trust:         trust,
		MaxAge:        source.MaxAge,
		CachePath:     source.CachePath,
		CacheFallback: source.CacheFallback,
		MaxBytes:      source.MaxBytes,
	})
}

func policyAppMode(mode app.Mode, source policySource) (app.Mode, error) {
	if mode == modePolicyTarget {
		if !source.configured() {
			return "", fmt.Errorf("%w: policy source is required for policy-target mode", policy.ErrInvalidPolicy)
		}
		return app.ModePlanTarget, nil
	}
	if !source.configured() {
		return mode, nil
	}
	switch mode {
	case app.ModeMenu:
		return app.ModeBootTarget, nil
	case app.ModePlanTarget, app.ModeStageTarget, app.ModeBootTarget:
		return mode, nil
	default:
		return "", fmt.Errorf("%w: dynamic policy requires policy-target, menu, plan-target, stage-target, or boot-target mode", policy.ErrInvalidPolicy)
	}
}

func policyTrust(source policySource) (policy.Trust, error) {
	if source.SignaturePath == "" || source.PublicKeyPath == "" {
		return policy.Trust{}, fmt.Errorf("%w: policy Ed25519 signature and public key paths are both required", policy.ErrInvalidPolicy)
	}
	signature, err := readHexOrRawFile(source.SignaturePath)
	if err != nil {
		return policy.Trust{}, fmt.Errorf("%w: read policy signature: %w", policy.ErrInvalidPolicy, err)
	}
	publicKey, err := readHexOrRawFile(source.PublicKeyPath)
	if err != nil {
		return policy.Trust{}, fmt.Errorf("%w: read policy public key: %w", policy.ErrInvalidPolicy, err)
	}
	return policy.Trust{
		Ed25519Signature: signature,
		Ed25519PublicKey: publicKey,
	}, nil
}

func diagnosticsPolicyPosture(source policySource) diagnostics.PolicyPosture {
	posture := diagnostics.PolicyPosture{
		LocalPathSet:   source.Path != "",
		RemoteURLSet:   source.URL != "",
		Ed25519:        source.SignaturePath != "" || source.PublicKeyPath != "",
		SignatureFiles: source.SignaturePath != "" || source.PublicKeyPath != "",
		Freshness:      source.configured(),
		Cache:          source.CachePath != "",
		CacheFallback:  source.CacheFallback,
		MaxBytes:       source.MaxBytes > 0,
	}
	if source.Fallback != "" && source.Fallback != policyFallbackNone {
		posture.Fallback = string(source.Fallback)
	}
	switch {
	case source.Path != "" && source.URL != "":
		posture.Source = "multiple"
	case source.URL != "":
		posture.Source = "remote"
	case source.Path != "":
		posture.Source = "local"
	}
	return posture
}

func policyPostureWithFallback(posture diagnostics.PolicyPosture, fallback string) diagnostics.PolicyPosture {
	posture.Fallback = fallback
	return posture
}

func policyPostureWithDecision(posture diagnostics.PolicyPosture, decision policy.Decision) diagnostics.PolicyPosture {
	posture.DecisionID = decision.DecisionID
	posture.TargetID = decision.TargetID
	if decision.PublishedAt != nil {
		posture.PublishedAt = decision.PublishedAt.Format(time.RFC3339)
	}
	if decision.ExpiresAt != nil {
		posture.ExpiresAt = decision.ExpiresAt.Format(time.RFC3339)
	}
	return posture
}

func diagnosticsPolicyRedactValues(source policySource) []string {
	values := []string{
		source.Path,
		source.URL,
		source.SignaturePath,
		source.PublicKeyPath,
		source.CachePath,
	}
	redactions := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			redactions = append(redactions, value)
		}
	}
	return redactions
}

type catalogSource struct {
	Path              string
	URL               string
	SHA256            string
	SignaturePath     string
	PublicKeyPath     string
	MaxAge            time.Duration
	RequireFreshness  bool
	CachePath         string
	CacheFallback     bool
	MaxBytes          int64
	IncludeDefault    bool
	CompiledProviders []string
}

func loadCatalog(ctx context.Context, source catalogSource) (catalog.Document, error) {
	if source.Path != "" && source.URL != "" {
		return catalog.Document{}, errors.New("cannot use --catalog with --catalog-url")
	}
	providerIDs := source.CompiledProviders
	if providerIDs == nil {
		providerIDs = compiledProviderIDs()
	}
	if source.Path == "" && source.URL == "" {
		return catalog.LoadDefault(providerIDs)
	}
	selected, err := loadSelectedCatalog(ctx, source, providerIDs)
	if err != nil {
		return catalog.Document{}, err
	}
	if !source.IncludeDefault {
		return selected, nil
	}
	defaultDoc, err := catalog.LoadDefault(providerIDs)
	if err != nil {
		return catalog.Document{}, err
	}
	return catalog.Compose(defaultDoc, selected)
}

func loadSelectedCatalog(ctx context.Context, source catalogSource, providerIDs []string) (catalog.Document, error) {
	if source.URL == "" {
		return catalog.LoadFile(source.Path, providerIDs)
	}
	trust, err := hostedTrust(source)
	if err != nil {
		return catalog.Document{}, err
	}
	return catalog.LoadHosted(ctx, catalog.HostedOptions{
		URL:              source.URL,
		Trust:            trust,
		ProviderIDs:      providerIDs,
		MaxAge:           source.MaxAge,
		RequireFreshness: source.RequireFreshness,
		CachePath:        source.CachePath,
		CacheFallback:    source.CacheFallback,
		MaxBytes:         source.MaxBytes,
	})
}

func hostedTrust(source catalogSource) (catalog.HostedTrust, error) {
	trust := catalog.HostedTrust{SHA256: strings.TrimSpace(source.SHA256)}
	if source.SignaturePath == "" && source.PublicKeyPath == "" {
		if trust.SHA256 == "" {
			return catalog.HostedTrust{}, errors.New("hosted catalog trust configuration is required")
		}
		return trust, nil
	}
	if source.SignaturePath == "" || source.PublicKeyPath == "" {
		return catalog.HostedTrust{}, errors.New("hosted catalog Ed25519 signature and public key paths are both required")
	}
	signature, err := readHexOrRawFile(source.SignaturePath)
	if err != nil {
		return catalog.HostedTrust{}, fmt.Errorf("read hosted catalog signature: %w", err)
	}
	publicKey, err := readHexOrRawFile(source.PublicKeyPath)
	if err != nil {
		return catalog.HostedTrust{}, fmt.Errorf("read hosted catalog public key: %w", err)
	}
	trust.Ed25519Signature = signature
	trust.Ed25519PublicKey = publicKey
	return trust, nil
}

func readHexOrRawFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(data)
	decoded, err := hex.DecodeString(string(trimmed))
	if err == nil && len(decoded) > 0 {
		return decoded, nil
	}
	return data, nil
}
