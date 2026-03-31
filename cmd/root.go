package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/dlukt/bmad-autopilot/internal/orchestrator"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	statusFile              string
	brain                   string
	workdir                 string
	copilotModel            string
	timeout                 time.Duration
	showCommandOutput       bool
	tui                     bool
	createStorySlashCommand string
	devStorySlashCommand    string
	codeReviewSlashCommand  string
	createStorySlashOptions string
	devStorySlashOptions    string
	codeReviewSlashOptions  string
}

// Execute runs the CLI entrypoint.
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	defaultSlashCommands := orchestrator.DefaultSlashCommandOptions()

	opts := &rootOptions{
		statusFile:              "",
		brain:                   "deterministic",
		workdir:                 "",
		timeout:                 0,
		showCommandOutput:       true,
		tui:                     true,
		createStorySlashCommand: defaultSlashCommands.CreateStory,
		devStorySlashCommand:    defaultSlashCommands.DevStory,
		codeReviewSlashCommand:  defaultSlashCommands.CodeReview,
		createStorySlashOptions: defaultSlashCommands.CreateStoryOptions,
		devStorySlashOptions:    defaultSlashCommands.DevStoryOptions,
		codeReviewSlashOptions:  defaultSlashCommands.CodeReviewOptions,
	}

	cmd := &cobra.Command{
		Use:   "bmad-autopilot",
		Short: "Manual loop runner for BMAD sprint stories",
	}

	cmd.PersistentFlags().StringVar(&opts.statusFile, "status-file", opts.statusFile, "Path to sprint-status.yaml (default: <cwd>/_bmad-output/implementation-artifacts/sprint-status.yaml)")
	cmd.PersistentFlags().StringVar(&opts.brain, "brain", opts.brain, "Overseer brain (default: deterministic; options: deterministic, glm-5)")
	cmd.PersistentFlags().StringVar(&opts.workdir, "workdir", opts.workdir, "Working directory for copilot/git operations (default: inferred from status file path)")
	cmd.PersistentFlags().StringVar(&opts.copilotModel, "copilot-model", opts.copilotModel, "Optional Copilot model override")
	cmd.PersistentFlags().DurationVar(&opts.timeout, "timeout", opts.timeout, "Per-command timeout (0 disables timeout)")
	cmd.PersistentFlags().BoolVar(&opts.showCommandOutput, "show-command-output", opts.showCommandOutput, "Print raw Copilot output for each command (default: true)")
	cmd.PersistentFlags().BoolVar(&opts.tui, "tui", opts.tui, "Enable Bubble Tea TUI in interactive terminals (default: true)")
	cmd.PersistentFlags().StringVar(&opts.createStorySlashCommand, "create-story-slash-command", opts.createStorySlashCommand, "Slash command to create stories (default: /bmad-create-story)")
	cmd.PersistentFlags().StringVar(&opts.devStorySlashCommand, "dev-story-slash-command", opts.devStorySlashCommand, "Slash command to develop stories (default: /bmad-dev-story)")
	cmd.PersistentFlags().StringVar(&opts.codeReviewSlashCommand, "code-review-slash-command", opts.codeReviewSlashCommand, "Slash command to review stories (default: /bmad-code-review)")
	cmd.PersistentFlags().StringVar(&opts.createStorySlashOptions, "create-story-slash-options", opts.createStorySlashOptions, "Extra options appended after story number for create-story slash command")
	cmd.PersistentFlags().StringVar(&opts.devStorySlashOptions, "dev-story-slash-options", opts.devStorySlashOptions, "Extra options appended after story number for dev-story slash command")
	cmd.PersistentFlags().StringVar(&opts.codeReviewSlashOptions, "code-review-slash-options", opts.codeReviewSlashOptions, "Extra options appended after story number for code-review slash command")

	cmd.AddCommand(newRunCmd(opts))
	return cmd
}
