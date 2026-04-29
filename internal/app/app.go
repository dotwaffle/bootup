// Package app coordinates bootup startup and operator flows.
package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/ui"
	"golang.org/x/term"
)

// Mode selects the startup behavior.
type Mode string

const (
	// ModeMenu prompts on the text interface, then stages and boots the chosen
	// target.
	ModeMenu Mode = "menu"

	// ModeListTargets prints targets and exits. It is useful for tests and
	// non-interactive diagnostics.
	ModeListTargets Mode = "list-targets"

	// ModePlanTarget selects TargetID and prints its boot plan without handoff.
	ModePlanTarget Mode = "plan-target"

	// ModeStageTarget selects TargetID, verifies artifacts, and stages them.
	ModeStageTarget Mode = "stage-target"

	// ModeBootTarget stages TargetID and executes the kexec handoff.
	ModeBootTarget Mode = "boot-target"
)

// UIMode selects the operator interface style for interactive menu mode.
type UIMode string

const (
	// UIModeAuto uses the rich UI only when stdin and stdout are terminals.
	UIModeAuto UIMode = "auto"

	// UIModeRich requires the rich terminal UI.
	UIModeRich UIMode = "rich"

	// UIModePlain always uses the plain text prompt.
	UIModePlain UIMode = "plain"
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

// Executor executes a staged boot plan.
type Executor interface {
	Execute(context.Context, provider.BootPlan) error
}

// Config contains immutable app startup dependencies.
type Config struct {
	Registry            *provider.Registry
	Stdin               io.Reader
	Stdout              io.Writer
	Stderr              io.Writer
	Logger              *slog.Logger
	Mode                Mode
	UIMode              UIMode
	TargetID            string
	StagingDir          string
	Hold                bool
	Executor            Executor
	Preparers           []Preparer
	OnBeforeListTargets func()
}

// App runs the bootup stage-1 flow.
type App struct {
	config Config
}

// New creates an App from config.
func New(config Config) *App {
	if config.Stdin == nil {
		config.Stdin = os.Stdin
	}
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
	if config.UIMode == "" {
		config.UIMode = UIModeAuto
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

	if err := a.runMode(ctx); err != nil {
		return err
	}
	if a.config.Hold {
		a.config.Logger.Info("bootup holding after mode completion")
		return hold(ctx)
	}
	return nil
}

func hold(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Hour):
		}
	}
}

func (a *App) runMode(ctx context.Context) error {
	switch a.config.Mode {
	case ModeMenu:
		return a.menu(ctx)
	case "", ModeListTargets:
		return a.listTargets(ctx)
	case ModePlanTarget:
		return a.planTarget(ctx)
	case ModeStageTarget:
		_, err := a.stageTarget(ctx, a.textMenu())
		return err
	case ModeBootTarget:
		return a.bootTarget(ctx)
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
	target, err := a.selectTarget(ctx)
	if err != nil {
		return err
	}

	menu := a.textMenu()
	if err := menu.RenderStatus(a.config.Stdout, "planning", target.Name); err != nil {
		return fmt.Errorf("render status: %w", err)
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

type statusRenderer interface {
	RenderStatus(io.Writer, string, string) error
	RenderFatal(io.Writer, string) error
}

func (a *App) stageTarget(ctx context.Context, renderer statusRenderer) (provider.BootPlan, error) {
	target, err := a.selectTarget(ctx)
	if err != nil {
		return provider.BootPlan{}, err
	}
	return a.stageSelectedTarget(ctx, target, renderer)
}

func (a *App) stageSelectedTarget(ctx context.Context, target provider.Target, renderer statusRenderer) (provider.BootPlan, error) {
	stagingDir := a.config.StagingDir
	if stagingDir == "" {
		stagingDir = "/tmp/bootup"
	}
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return provider.BootPlan{}, fmt.Errorf("create staging dir: %w", err)
	}

	if err := renderer.RenderStatus(a.config.Stdout, "planning", target.Name); err != nil {
		return provider.BootPlan{}, fmt.Errorf("render status: %w", err)
	}
	plan, err := a.config.Registry.Plan(ctx, target)
	if err != nil {
		if renderErr := renderer.RenderFatal(a.config.Stdout, err.Error()); renderErr != nil {
			return provider.BootPlan{}, fmt.Errorf("render fatal error: %w", renderErr)
		}
		return provider.BootPlan{}, err
	}
	if err := renderer.RenderStatus(a.config.Stdout, "verifying", target.Name); err != nil {
		return provider.BootPlan{}, fmt.Errorf("render status: %w", err)
	}
	if err := renderer.RenderStatus(a.config.Stdout, "staging", target.Name); err != nil {
		return provider.BootPlan{}, fmt.Errorf("render status: %w", err)
	}
	staged, err := a.config.Registry.Stage(ctx, provider.StageConfig{
		Plan:       plan,
		StagingDir: stagingDir,
	})
	if err != nil {
		if renderErr := renderer.RenderFatal(a.config.Stdout, err.Error()); renderErr != nil {
			return provider.BootPlan{}, fmt.Errorf("render fatal error: %w", renderErr)
		}
		return provider.BootPlan{}, err
	}
	if _, err := fmt.Fprintf(a.config.Stdout, "kernel\t%s\ninitrd\t%s\ncmdline\t%s\n", staged.Kernel.Path, staged.Initrd.Path, staged.Cmdline); err != nil {
		return provider.BootPlan{}, fmt.Errorf("write staged boot plan: %w", err)
	}
	return staged, nil
}

func (a *App) bootTarget(ctx context.Context) error {
	menu := a.textMenu()
	staged, err := a.stageTarget(ctx, menu)
	if err != nil {
		return err
	}
	return a.executeStaged(ctx, staged, menu)
}

func (a *App) executeStaged(ctx context.Context, staged provider.BootPlan, renderer statusRenderer) error {
	executor := a.config.Executor
	if executor == nil {
		return errors.New("executor is required")
	}
	if err := renderer.RenderStatus(a.config.Stdout, "loading", staged.Target.Name); err != nil {
		return fmt.Errorf("render status: %w", err)
	}
	if err := executor.Execute(ctx, staged); err != nil {
		if renderErr := renderer.RenderFatal(a.config.Stdout, err.Error()); renderErr != nil {
			return fmt.Errorf("render fatal error: %w", renderErr)
		}
		return err
	}
	return nil
}

func (a *App) menu(ctx context.Context) error {
	targets, err := a.config.Registry.Targets(ctx)
	if err != nil {
		return fmt.Errorf("list targets: %w", err)
	}
	useRich, err := a.useRichMenu()
	if err != nil {
		return err
	}
	if useRich {
		menu := a.richMenu()
		target, err := menu.SelectTarget(ctx, targets)
		if err != nil {
			if errors.Is(err, ui.ErrSelectionCanceled) {
				return nil
			}
			return err
		}
		staged, err := a.stageSelectedTarget(ctx, target, menu)
		if err != nil {
			return err
		}
		return a.executeStaged(ctx, staged, menu)
	}
	return a.plainMenu(ctx, targets)
}

func (a *App) plainMenu(ctx context.Context, targets []provider.Target) error {
	menu := a.textMenu()
	if err := menu.RenderTargets(a.config.Stdout, targets); err != nil {
		return fmt.Errorf("render targets: %w", err)
	}
	if err := menu.RenderPrompt(a.config.Stdout, "target> "); err != nil {
		return err
	}

	input, err := bufio.NewReader(a.config.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read target selection: %w", err)
	}
	target, err := ui.SelectTargetByInput(targets, input)
	if err != nil {
		if renderErr := menu.RenderFatal(a.config.Stdout, err.Error()); renderErr != nil {
			return fmt.Errorf("render fatal error: %w", renderErr)
		}
		return err
	}
	staged, err := a.stageSelectedTarget(ctx, target, menu)
	if err != nil {
		return err
	}
	return a.executeStaged(ctx, staged, menu)
}

func (a *App) selectTarget(ctx context.Context) (provider.Target, error) {
	targets, err := a.config.Registry.Targets(ctx)
	if err != nil {
		return provider.Target{}, fmt.Errorf("list targets: %w", err)
	}
	return ui.SelectTargetByID(targets, a.config.TargetID)
}

func (a *App) textMenu() ui.TextMenu {
	return ui.TextMenu{Width: 80}
}

func (a *App) richMenu() ui.RichMenu {
	return ui.RichMenu{
		Width:   80,
		Stdin:   a.config.Stdin,
		Stdout:  a.config.Stdout,
		Animate: true,
	}
}

func (a *App) useRichMenu() (bool, error) {
	switch a.config.UIMode {
	case UIModeAuto:
		return a.hasInteractiveTerminal(), nil
	case UIModeRich:
		if !a.hasInteractiveTerminal() {
			return false, errors.New("rich UI requires terminal stdin and stdout")
		}
		return true, nil
	case UIModePlain:
		return false, nil
	default:
		return false, fmt.Errorf("unsupported ui mode %q", a.config.UIMode)
	}
}

func (a *App) hasInteractiveTerminal() bool {
	stdin, ok := a.config.Stdin.(*os.File)
	if !ok {
		return false
	}
	stdout, ok := a.config.Stdout.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(stdin.Fd())) && term.IsTerminal(int(stdout.Fd()))
}
