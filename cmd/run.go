package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/dlukt/bmad-autopilot/internal/orchestrator"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newRunCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run one-story-at-a-time manual loop",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if shouldUseTUI(opts) {
				return runWithTUI(cmd, opts)
			}
			if opts.tui {
				fmt.Fprintln(cmd.OutOrStdout(), "TUI: disabled because stdin/stdout are not interactive terminals")
			}
			return runWithoutTUI(cmd, opts)
		},
	}
}

func runWithoutTUI(cmd *cobra.Command, opts *rootOptions) error {
	runCtx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	stopController := orchestrator.NewStopController()
	stopHotkeys, err := startStopHotkeys(stopController, cancel, cmd.OutOrStdout())
	if err != nil {
		return err
	}
	hotkeysClosed := false
	defer func() {
		if !hotkeysClosed {
			stopHotkeys()
		}
	}()

	outcome, err := runOrchestrator(runCtx, opts, stopController, cmd.OutOrStdout())
	if err != nil {
		return err
	}

	stopHotkeys()
	hotkeysClosed = true

	if outcome == orchestrator.RunOutcomeCompleted && stopController.PoweroffArmed() {
		fmt.Fprintln(cmd.OutOrStdout(), "POWEROFF: armed and loop completed; running `sudo systemctl poweroff`")
		if err := runSystemPoweroff(runCtx, cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
			return err
		}
	}

	return nil
}

func runOrchestrator(
	ctx context.Context,
	opts *rootOptions,
	stopChecker orchestrator.StopChecker,
	output io.Writer,
) (orchestrator.RunOutcome, error) {
	runner, err := orchestrator.New(orchestrator.Config{
		StatusFile:           opts.statusFile,
		Brain:                opts.brain,
		Workdir:              opts.workdir,
		CopilotModel:         opts.copilotModel,
		CommandTimeout:       opts.timeout,
		DisableCommandOutput: !opts.showCommandOutput,
		StopChecker:          stopChecker,
		Output:               output,
	})
	if err != nil {
		return orchestrator.RunOutcomeUnknown, err
	}
	return runner.Run(ctx)
}

func shouldUseTUI(opts *rootOptions) bool {
	if !opts.tui {
		return false
	}
	return isTerminal(os.Stdin) && isTerminal(os.Stdout)
}

func isTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}
