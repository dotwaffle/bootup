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
	"strings"
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

	// ModeDiscoverTargets discovers concrete targets for one provider family
	// and exits. It is useful for non-interactive diagnostics.
	ModeDiscoverTargets Mode = "discover-targets"

	// ModeShowTarget prints detailed metadata for TargetID and exits.
	ModeShowTarget Mode = "show-target"

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
	bootConsolePath = "/dev/console"
	uRootBusybox    = "/bbin/bb"

	// UIModeAuto uses the rich UI when stdin and stdout look console-backed.
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
	TargetOptions       []provider.SelectedOption
	DiscoveryFamilyID   string
	StagingDir          string
	CmdlineAppend       string
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
	case ModeDiscoverTargets:
		return a.discoverTargets(ctx)
	case ModeShowTarget:
		return a.showTarget(ctx)
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

func (a *App) showTarget(ctx context.Context) error {
	target, err := a.selectTarget(ctx)
	if err != nil {
		return err
	}
	menu := ui.TextMenu{Width: 120}
	if err := menu.RenderTargetDetails(a.config.Stdout, target); err != nil {
		return fmt.Errorf("render target details: %w", err)
	}
	return nil
}

func (a *App) discoverTargets(ctx context.Context) error {
	familyID := a.config.DiscoveryFamilyID
	if strings.TrimSpace(familyID) == "" {
		return errors.New("discovery family is required")
	}
	targets, err := a.config.Registry.DiscoverTargets(ctx, familyID)
	if err != nil {
		return fmt.Errorf("discover targets: %w", err)
	}
	if len(targets) == 0 {
		if _, writeErr := fmt.Fprintf(a.config.Stdout, "no discovered targets for %s\n", familyID); writeErr != nil {
			return fmt.Errorf("write empty discovery result: %w", writeErr)
		}
		return fmt.Errorf("no targets discovered for %s", familyID)
	}
	menu := ui.TextMenu{Width: 80}
	if err := menu.RenderTargets(a.config.Stdout, targets); err != nil {
		return fmt.Errorf("render discovered targets: %w", err)
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
	plan, err := a.config.Registry.Plan(ctx, provider.PlanInput{
		Target:  target,
		Options: a.config.TargetOptions,
	})
	if err != nil {
		if renderErr := menu.RenderFatal(a.config.Stdout, err.Error()); renderErr != nil {
			return fmt.Errorf("render fatal error: %w", renderErr)
		}
		return err
	}
	plan = a.applyCmdlineAppend(plan)
	if err := writeBootPlan(a.config.Stdout, plan); err != nil {
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
	plan, err := a.config.Registry.Plan(ctx, provider.PlanInput{
		Target:  target,
		Options: a.config.TargetOptions,
	})
	if err != nil {
		if renderErr := renderer.RenderFatal(a.config.Stdout, err.Error()); renderErr != nil {
			return provider.BootPlan{}, fmt.Errorf("render fatal error: %w", renderErr)
		}
		return provider.BootPlan{}, err
	}
	plan = a.applyCmdlineAppend(plan)
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
	if err := writeBootPlan(a.config.Stdout, staged); err != nil {
		return provider.BootPlan{}, fmt.Errorf("write staged boot plan: %w", err)
	}
	return staged, nil
}

func writeBootPlan(w io.Writer, plan provider.BootPlan) error {
	if plan.ResolvedAction() == provider.BootActionFreeBSDKboot {
		return writeFreeBSDKbootPlan(w, plan.FreeBSDKboot)
	}
	if _, err := fmt.Fprintf(w, "kernel\t%s\ninitrd\t%s\ncmdline\t%s\n", artifactLocation(plan.Kernel), artifactLocation(plan.Initrd), plan.Cmdline); err != nil {
		return err
	}
	return nil
}

func writeFreeBSDKbootPlan(w io.Writer, plan provider.FreeBSDKbootPlan) error {
	for _, line := range []struct {
		label string
		value string
	}{
		{label: "loader", value: artifactLocation(plan.Loader)},
		{label: "loader_help", value: artifactLocation(plan.LoaderHelp)},
		{label: "loader_archive", value: artifactLocation(plan.LoaderArchive)},
		{label: "payload", value: artifactLocation(plan.Payload)},
		{label: "payload_root", value: plan.PayloadRoot},
		{label: "args", value: strings.Join(plan.Args, " ")},
	} {
		if line.value == "" {
			continue
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\n", line.label, line.value); err != nil {
			return err
		}
	}
	return nil
}

func artifactLocation(artifact provider.Artifact) string {
	if artifact.Path != "" {
		return artifact.Path
	}
	return artifact.URL
}

func (a *App) applyCmdlineAppend(plan provider.BootPlan) provider.BootPlan {
	plan.Cmdline = appendCmdline(plan.Cmdline, a.config.CmdlineAppend)
	return plan
}

func appendCmdline(base string, extra string) string {
	base = strings.TrimSpace(base)
	extra = strings.TrimSpace(extra)
	switch {
	case base == "":
		return extra
	case extra == "":
		return base
	default:
		return base + " " + extra
	}
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
	families, err := a.config.Registry.DiscoveryFamilies()
	if err != nil {
		return fmt.Errorf("list discovery families: %w", err)
	}
	useRich, err := a.useRichMenu()
	if err != nil {
		return err
	}
	if useRich {
		menu := a.richMenu()
		option, err := menu.SelectBootOption(ctx, ui.BootOptions(targets, families))
		if err != nil {
			if errors.Is(err, ui.ErrSelectionCanceled) {
				return nil
			}
			return err
		}
		target, err := a.richTargetFromOption(ctx, option, menu)
		if err != nil {
			return err
		}
		staged, err := a.stageSelectedTarget(ctx, target, menu)
		if err != nil {
			return err
		}
		return a.executeStaged(ctx, staged, menu)
	}
	return a.plainMenu(ctx, targets, families)
}

func (a *App) richTargetFromOption(ctx context.Context, option ui.BootOption, menu ui.RichMenu) (provider.Target, error) {
	switch option.Kind {
	case ui.BootOptionTarget:
		return option.Target, nil
	case ui.BootOptionDiscoveryFamily:
		if err := menu.RenderStatus(a.config.Stdout, "discovering", option.Family.Name); err != nil {
			return provider.Target{}, fmt.Errorf("render status: %w", err)
		}
		targets, err := a.config.Registry.DiscoverTargets(ctx, option.Family.ID)
		if err != nil {
			if renderErr := menu.RenderFatal(a.config.Stdout, err.Error()); renderErr != nil {
				return provider.Target{}, fmt.Errorf("render fatal error: %w", renderErr)
			}
			return provider.Target{}, err
		}
		if len(targets) == 0 {
			err := fmt.Errorf("no targets discovered for %s", option.Family.ID)
			if renderErr := menu.RenderFatal(a.config.Stdout, err.Error()); renderErr != nil {
				return provider.Target{}, fmt.Errorf("render fatal error: %w", renderErr)
			}
			return provider.Target{}, err
		}
		target, err := menu.SelectTarget(ctx, targets)
		if err != nil {
			if errors.Is(err, ui.ErrSelectionCanceled) {
				return provider.Target{}, err
			}
			return provider.Target{}, err
		}
		return target, nil
	default:
		return provider.Target{}, fmt.Errorf("unsupported boot option kind %q", option.Kind)
	}
}

func (a *App) plainMenu(ctx context.Context, targets []provider.Target, families []provider.DiscoveryFamily) error {
	menu := a.textMenu()
	options := ui.BootOptions(targets, families)
	if err := menu.RenderBootOptions(a.config.Stdout, options); err != nil {
		return fmt.Errorf("render boot options: %w", err)
	}
	if err := menu.RenderPrompt(a.config.Stdout, "target> "); err != nil {
		return err
	}

	reader := bufio.NewReader(a.config.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read target selection: %w", err)
	}
	option, err := ui.SelectBootOptionByInput(options, input)
	if err != nil {
		if renderErr := menu.RenderFatal(a.config.Stdout, err.Error()); renderErr != nil {
			return fmt.Errorf("render fatal error: %w", renderErr)
		}
		return err
	}
	target, err := a.targetFromOption(ctx, option, menu, reader)
	if err != nil {
		return err
	}
	staged, err := a.stageSelectedTarget(ctx, target, menu)
	if err != nil {
		return err
	}
	return a.executeStaged(ctx, staged, menu)
}

func (a *App) targetFromOption(ctx context.Context, option ui.BootOption, menu ui.TextMenu, reader *bufio.Reader) (provider.Target, error) {
	switch option.Kind {
	case ui.BootOptionTarget:
		return option.Target, nil
	case ui.BootOptionDiscoveryFamily:
		if err := menu.RenderStatus(a.config.Stdout, "discovering", option.Family.Name); err != nil {
			return provider.Target{}, fmt.Errorf("render status: %w", err)
		}
		targets, err := a.config.Registry.DiscoverTargets(ctx, option.Family.ID)
		if err != nil {
			if renderErr := menu.RenderFatal(a.config.Stdout, err.Error()); renderErr != nil {
				return provider.Target{}, fmt.Errorf("render fatal error: %w", renderErr)
			}
			return provider.Target{}, err
		}
		if len(targets) == 0 {
			err := fmt.Errorf("no targets discovered for %s", option.Family.ID)
			if renderErr := menu.RenderFatal(a.config.Stdout, err.Error()); renderErr != nil {
				return provider.Target{}, fmt.Errorf("render fatal error: %w", renderErr)
			}
			return provider.Target{}, err
		}
		if err := menu.RenderTargets(a.config.Stdout, targets); err != nil {
			return provider.Target{}, fmt.Errorf("render discovered targets: %w", err)
		}
		if err := menu.RenderPrompt(a.config.Stdout, "target> "); err != nil {
			return provider.Target{}, err
		}
		input, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return provider.Target{}, fmt.Errorf("read target selection: %w", err)
		}
		target, err := ui.SelectTargetByInput(targets, input)
		if err != nil {
			if renderErr := menu.RenderFatal(a.config.Stdout, err.Error()); renderErr != nil {
				return provider.Target{}, fmt.Errorf("render fatal error: %w", renderErr)
			}
			return provider.Target{}, err
		}
		return target, nil
	default:
		return provider.Target{}, fmt.Errorf("unsupported boot option kind %q", option.Kind)
	}
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
		if a.hasInteractiveConsole() {
			return true, nil
		}
		if a.attachBootConsole() && a.hasInteractiveConsole() {
			return true, nil
		}
		return false, nil
	case UIModeRich:
		if !a.hasInteractiveConsole() {
			a.attachBootConsole()
		}
		if !a.hasInteractiveConsole() {
			return false, errors.New("rich UI requires console stdin and stdout")
		}
		return true, nil
	case UIModePlain:
		return false, nil
	default:
		return false, fmt.Errorf("unsupported ui mode %q", a.config.UIMode)
	}
}

func (a *App) attachBootConsole() bool {
	if !a.shouldAttachBootConsole() {
		return false
	}
	console, err := os.OpenFile(bootConsolePath, os.O_RDWR, 0)
	if err != nil {
		return false
	}
	a.config.Stdin = console
	a.config.Stdout = console
	return true
}

func (a *App) shouldAttachBootConsole() bool {
	stdin, stdinOK := a.config.Stdin.(*os.File)
	stdout, stdoutOK := a.config.Stdout.(*os.File)
	if !stdinOK || !stdoutOK {
		return false
	}
	if stdin != os.Stdin || stdout != os.Stdout {
		return false
	}
	_, err := os.Stat(uRootBusybox)
	return err == nil
}

func (a *App) hasInteractiveConsole() bool {
	stdin, ok := a.config.Stdin.(*os.File)
	if !ok {
		return false
	}
	stdout, ok := a.config.Stdout.(*os.File)
	if !ok {
		return false
	}
	if term.IsTerminal(int(stdin.Fd())) && term.IsTerminal(int(stdout.Fd())) {
		return true
	}
	return isConsoleLikeFile(stdin) && isConsoleLikeFile(stdout)
}

func isConsoleLikeFile(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	modeType := info.Mode().Type()
	if modeType&os.ModeCharDevice != 0 {
		return true
	}
	if modeType == 0 || modeType&os.ModeNamedPipe != 0 || modeType&os.ModeSocket != 0 {
		return false
	}
	if modeType&os.ModeDir != 0 || modeType&os.ModeDevice != 0 {
		return false
	}
	// Early boot consoles can be backed by unusual fd types before termios
	// reports a normal TTY. Treat those as interactive unless they are obvious
	// redirection targets handled above.
	return true
}
