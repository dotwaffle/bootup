package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/dotwaffle/bootup/internal/app"
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

	mode := flags.String("mode", string(app.ModeListTargets), "startup mode")
	uiMode := flags.String("ui", string(app.UIModeAuto), "menu UI mode: auto, rich, plain")
	targetID := flags.String("target", "", "target ID for non-interactive modes")
	discoveryFamilyID := flags.String("discovery-family", "", "discovery family ID for discover-targets mode")
	stagingDir := flags.String("staging-dir", "/tmp/bootup", "directory for verified boot artifacts")
	catalogPath := flags.String("catalog", "", "static provider catalog JSON path")
	providerConfigPath := flags.String("provider-config", "", "provider runtime config JSON path")
	hold := flags.Bool("hold", false, "wait after the selected mode completes")
	prepareRuntime := flags.Bool("prepare-runtime", false, "validate network, CA roots, and time before provider operations")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	var providerConfig providerconfig.Config
	if *providerConfigPath != "" {
		config, err := providerconfig.LoadFile(*providerConfigPath)
		if err != nil {
			return fmt.Errorf("load provider config: %w", err)
		}
		providerConfig = config
	}

	catalogDoc, err := loadCatalog(*catalogPath)
	if err != nil {
		return fmt.Errorf("load catalog: %w", err)
	}

	var preparers []app.Preparer
	if *prepareRuntime {
		preparers = append(preparers,
			runtime.NetworkPreparer{},
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
		DiscoveryFamilyID: *discoveryFamilyID,
		StagingDir:        *stagingDir,
		Hold:              *hold,
		Executor:          handoff.KexecExecutor{},
		Preparers:         preparers,
	})
	return runner.Run(ctx)
}

func loadCatalog(path string) (catalog.Document, error) {
	if path == "" {
		return catalog.LoadDefault(compiledProviderIDs())
	}
	return catalog.LoadFile(path, compiledProviderIDs())
}
