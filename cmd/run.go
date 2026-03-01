package cmd

import (
	"context"
	"fmt"

	"github.com/dlukt/bmad-autopilot/internal/orchestrator"

	"github.com/spf13/cobra"
)

func newRunCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run one-story-at-a-time manual loop",
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			runner, err := orchestrator.New(orchestrator.Config{
				StatusFile:           opts.statusFile,
				Brain:                opts.brain,
				Workdir:              opts.workdir,
				CopilotModel:         opts.copilotModel,
				CommandTimeout:       opts.timeout,
				DisableCommandOutput: !opts.showCommandOutput,
				StopChecker:          stopController,
			})
			if err != nil {
				return err
			}
			outcome, err := runner.Run(runCtx)
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
		},
	}
}
