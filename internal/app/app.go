// Package app coordinates bootup startup and operator flows.
package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/ui"
)

// Mode selects the startup behavior.
type Mode string

const (
	// ModeListTargets prints targets and exits. It is useful for tests and
	// non-interactive diagnostics.
	ModeListTargets Mode = "list-targets"

	// ModePlanTarget selects TargetID and prints its boot plan without handoff.
	ModePlanTarget Mode = "plan-target"
)

// Preparer prepares runtime state before provider operations.
type Preparer interface {
	Prepare(context.Context) error
}

// PrepareFunc adapts a function into a Preparer.
type PrepareFunc func(context.Context) error

// Prepare calls f(ctx).
func (f PrepareFunc) Prepare(ctx context.Context) error {
	return f(ctx)
}

// Config contains immutable app startup dependencies.
type Config struct {
	Registry            *provider.Registry
	Stdout              io.Writer
	Stderr              io.Writer
	Logger              *slog.Logger
	Mode                Mode
	TargetID            string
	Preparers           []Preparer
	OnBeforeListTargets func()
}

// App runs the bootup stage-1 flow.
type App struct {
	config Config
}

// New creates an App from config.
func New(config Config) *App {
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Logger == nil {
		config.Logger = slog.New(slog.NewTextHandler(config.Stderr, nil))
	}
	if config.Registry == nil {
		config.Registry = provider.NewRegistry()
	}
	return &App{config: config}
}

// Run starts the configured bootup flow.
func (a *App) Run(ctx context.Context) error {
	a.config.Logger.Info("bootup started", slog.String("mode", string(a.config.Mode)))
	for _, preparer := range a.config.Preparers {
		if err := preparer.Prepare(ctx); err != nil {
			return fmt.Errorf("prepare runtime: %w", err)
		}
	}

	switch a.config.Mode {
	case "", ModeListTargets:
		return a.listTargets(ctx)
	case ModePlanTarget:
		return a.planTarget(ctx)
	default:
		return fmt.Errorf("unsupported mode %q", a.config.Mode)
	}
}

func (a *App) listTargets(ctx context.Context) error {
	if a.config.OnBeforeListTargets != nil {
		a.config.OnBeforeListTargets()
	}
	targets, err := a.config.Registry.Targets(ctx)
	if err != nil {
		return fmt.Errorf("list targets: %w", err)
	}
	menu := ui.TextMenu{Width: 80}
	if err := menu.RenderTargets(a.config.Stdout, targets); err != nil {
		return fmt.Errorf("render targets: %w", err)
	}
	return nil
}

func (a *App) planTarget(ctx context.Context) error {
	targets, err := a.config.Registry.Targets(ctx)
	if err != nil {
		return fmt.Errorf("list targets: %w", err)
	}
	target, err := ui.SelectTargetByID(targets, a.config.TargetID)
	if err != nil {
		return err
	}

	menu := ui.TextMenu{Width: 80}
	if err := menu.RenderProgress(a.config.Stdout, "planning "+target.Name); err != nil {
		return fmt.Errorf("render progress: %w", err)
	}
	plan, err := a.config.Registry.Plan(ctx, target)
	if err != nil {
		if renderErr := menu.RenderFatal(a.config.Stdout, err.Error()); renderErr != nil {
			return fmt.Errorf("render fatal error: %w", renderErr)
		}
		return err
	}
	if _, err := fmt.Fprintf(a.config.Stdout, "kernel\t%s\ninitrd\t%s\ncmdline\t%s\n", plan.Kernel.URL, plan.Initrd.URL, plan.Cmdline); err != nil {
		return fmt.Errorf("write boot plan: %w", err)
	}
	return nil
}
