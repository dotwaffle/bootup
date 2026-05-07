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
	"github.com/dotwaffle/bootup/internal/handoff"
	"github.com/dotwaffle/bootup/internal/logging"
	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providerconfig"
	"github.com/dotwaffle/bootup/internal/runtime"

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
	providerConfigPath := flags.String("provider-config", "", "provider runtime config JSON path")
	cmdlineAppend := flags.String("append-cmdline", "", "additional kernel command-line parameters for selected targets")
	var targetOptions optionFlags
	flags.Var(&targetOptions, "option", "target option selection as id=value; repeatable")
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

	var providerConfig providerconfig.Config
	if *providerConfigPath != "" {
		config, err := providerconfig.LoadFile(*providerConfigPath)
		if err != nil {
			return fmt.Errorf("load provider config: %w", err)
		}
		providerConfig = config
	}

	catalogDoc, err := loadCatalog(ctx, catalogSource{
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
		CompiledProviders: compiledProviderIDs(),
	})
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
		preparers = append(preparers, runtime.NetworkPreparer{Config: networkConfig})
	}
	if *prepareRuntime {
		preparers = append(preparers,
			app.PrepareFunc(func(ctx context.Context) error {
				return runtime.CertPreparer{}.Prepare()
			}),
			runtime.TimePreparer{},
		)
	}

	registry := provider.NewRegistry()
	if err := registerProviders(registry, providerConfig, catalogDoc); err != nil {
		return err
	}

	runner := app.New(app.Config{
		Registry:          registry,
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		Logger:            logging.NewSerialLogger(stderr, slog.LevelInfo),
		Mode:              app.Mode(*mode),
		UIMode:            app.UIMode(*uiMode),
		TargetID:          *targetID,
		TargetOptions:     []provider.SelectedOption(targetOptions),
		DiscoveryFamilyID: *discoveryFamilyID,
		StagingDir:        *stagingDir,
		CmdlineAppend:     *cmdlineAppend,
		Hold:              *hold,
		Executor:          handoff.KexecExecutor{},
		Preparers:         preparers,
	})
	return runner.Run(ctx)
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
	if source.URL == "" {
		if source.Path == "" {
			return catalog.LoadDefault(providerIDs)
		}
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
