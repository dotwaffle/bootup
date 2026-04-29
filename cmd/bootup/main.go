package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/dotwaffle/bootup/internal/app"
	"github.com/dotwaffle/bootup/internal/handoff"
	"github.com/dotwaffle/bootup/internal/logging"
	"github.com/dotwaffle/bootup/internal/provider"
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
	flags := flag.NewFlagSet("bootup", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	mode := flags.String("mode", string(app.ModeListTargets), "startup mode")
	uiMode := flags.String("ui", string(app.UIModeAuto), "menu UI mode: auto, rich, plain")
	targetID := flags.String("target", "", "target ID for non-interactive modes")
	stagingDir := flags.String("staging-dir", "/tmp/bootup", "directory for verified boot artifacts")
	hold := flags.Bool("hold", false, "wait after the selected mode completes")
	prepareRuntime := flags.Bool("prepare-runtime", false, "validate network, CA roots, and time before provider operations")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
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
	if err := registerProviders(registry); err != nil {
		return err
	}

	runner := app.New(app.Config{
		Registry:   registry,
		Logger:     logging.NewSerialLogger(os.Stderr, slog.LevelInfo),
		Mode:       app.Mode(*mode),
		UIMode:     app.UIMode(*uiMode),
		TargetID:   *targetID,
		StagingDir: *stagingDir,
		Hold:       *hold,
		Executor:   handoff.KexecExecutor{},
		Preparers:  preparers,
	})
	return runner.Run(ctx)
}
