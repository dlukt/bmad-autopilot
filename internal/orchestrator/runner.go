package orchestrator

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dlukt/bmad-autopilot/internal/brain"
)

type Config struct {
	StatusFile           string
	Brain                string
	Workdir              string
	CopilotModel         string
	CommandTimeout       time.Duration
	DisableCommandOutput bool
	StopChecker          StopChecker
	Output               io.Writer
}

type Runner struct {
	cfg      Config
	brain    brain.Brain
	executor CommandExecutor
	stop     StopChecker
	output   io.Writer
}

const defaultStatusFile = "_bmad-output/implementation-artifacts/sprint-status.yaml"

type RunOutcome int

const (
	RunOutcomeUnknown RunOutcome = iota
	RunOutcomeCompleted
	RunOutcomeStopped
)

func New(cfg Config) (*Runner, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve current directory: %w", err)
	}

	statusFile, err := resolveStatusFilePath(cfg.StatusFile, cwd)
	if err != nil {
		return nil, err
	}
	cfg.StatusFile = statusFile

	if strings.TrimSpace(cfg.Workdir) == "" {
		cfg.Workdir = inferWorkdirFromStatusFile(cfg.StatusFile, cwd)
	}

	absWorkdir, err := filepath.Abs(cfg.Workdir)
	if err != nil {
		return nil, fmt.Errorf("resolve workdir: %w", err)
	}
	cfg.Workdir = absWorkdir

	selectedBrain, err := brain.New(cfg.Brain)
	if err != nil {
		return nil, err
	}
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}
	cfg.Output = output

	return &Runner{
		cfg:   cfg,
		brain: selectedBrain,
		executor: NewSDKExecutor(cfg.Workdir, cfg.CopilotModel, SDKExecutorOptions{
			Output:     cfg.Output,
			LiveOutput: !cfg.DisableCommandOutput,
		}),
		stop:   withDefaultStopChecker(cfg.StopChecker),
		output: cfg.Output,
	}, nil
}

func (r *Runner) Run(ctx context.Context) (RunOutcome, error) {
	for {
		if r.stopRequested() {
			return RunOutcomeStopped, nil
		}

		sprintStatus, err := LoadSprintStatus(r.cfg.StatusFile)
		if err != nil {
			return RunOutcomeUnknown, err
		}

		story, ok := sprintStatus.NextPendingStory()
		if !ok {
			r.logf("DONE: all non-retrospective stories are done\n")
			return RunOutcomeCompleted, nil
		}

		storyNumber, err := StoryNumberFromKey(story.Key)
		if err != nil {
			return RunOutcomeUnknown, err
		}

		primaryActions, err := PlanPrimaryActions(story.Status, storyNumber)
		if err != nil {
			return RunOutcomeUnknown, err
		}

		for _, action := range primaryActions {
			if r.stopRequested() {
				return RunOutcomeStopped, nil
			}
			if _, _, err := r.runStep(ctx, story.Key, action); err != nil {
				return RunOutcomeUnknown, err
			}
			if r.stopRequested() {
				return RunOutcomeStopped, nil
			}
		}

		reviewAction := ReviewAction(storyNumber)
		for {
			if r.stopRequested() {
				return RunOutcomeStopped, nil
			}
			result, afterStatus, err := r.runStep(ctx, story.Key, reviewAction)
			if err != nil {
				return RunOutcomeUnknown, err
			}
			if r.stopRequested() {
				return RunOutcomeStopped, nil
			}
			if !ShouldContinueReview(afterStatus, result.Published) {
				break
			}
		}
	}
}

func (r *Runner) stopRequested() bool {
	if r.stop.ShouldStop() {
		r.logf("STOP: graceful stop requested; exiting loop\n")
		return true
	}
	return false
}

func (r *Runner) runStep(ctx context.Context, storyKey string, action Action) (ExecResult, string, error) {
	beforeStatus, err := r.statusForStory(storyKey)
	if err != nil {
		return ExecResult{}, "", err
	}
	r.logf("STORY: %s\n", storyKey)
	r.logf("STATUS(before): %s\n", beforeStatus)
	r.logf("ACTION: %s\n", action.Command)

	commandCtx := ctx
	cancel := func() {}
	if r.cfg.CommandTimeout > 0 {
		commandCtx, cancel = context.WithTimeout(ctx, r.cfg.CommandTimeout)
	}
	defer cancel()

	execResult, execErr := r.executor.Run(commandCtx, action)
	if !r.cfg.DisableCommandOutput {
		if execResult.LiveOutput {
			r.logf("OUTPUT: <streamed live>\n")
		} else {
			r.printRawOutput(execResult.RawOutput)
		}
	}

	resultLine := r.summarizeResult(commandCtx, action.Command, execResult.RawOutput)
	if execErr != nil {
		if resultLine == "" {
			resultLine = execErr.Error()
		} else {
			resultLine = fmt.Sprintf("%s; error: %v", resultLine, execErr)
		}
	}
	r.logf("RESULT: %s\n", oneLine(resultLine))

	afterStatus, err := r.statusForStory(storyKey)
	if err != nil {
		return execResult, "", err
	}
	r.logf("STATUS(after): %s\n", afterStatus)

	if execErr != nil {
		return execResult, afterStatus, execErr
	}
	return execResult, afterStatus, nil
}

func (r *Runner) statusForStory(storyKey string) (string, error) {
	current, err := LoadSprintStatus(r.cfg.StatusFile)
	if err != nil {
		return "", err
	}

	status, ok := current.StoryStatus(storyKey)
	if !ok {
		return "<missing>", nil
	}
	return status, nil
}

func (r *Runner) summarizeResult(ctx context.Context, command string, rawOutput string) string {
	summary, err := r.brain.Summarize(ctx, command, rawOutput)
	if err != nil || strings.TrimSpace(summary) == "" {
		return fallbackSummary(rawOutput)
	}
	return summary
}

func fallbackSummary(rawOutput string) string {
	lines := strings.Split(rawOutput, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}
	return "no output"
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func (r *Runner) printRawOutput(rawOutput string) {
	trimmed := strings.TrimSpace(rawOutput)
	if trimmed == "" {
		r.logf("OUTPUT: <no output>\n")
		return
	}
	r.logf("OUTPUT:\n")
	r.logf("%s\n", trimmed)
}

func (r *Runner) logf(format string, args ...any) {
	out := r.output
	if out == nil {
		out = os.Stdout
	}
	fmt.Fprintf(out, format, args...)
}

func resolveStatusFilePath(statusFile, cwd string) (string, error) {
	statusFile = strings.TrimSpace(statusFile)
	if statusFile == "" {
		statusFile = defaultStatusFile
	}
	if filepath.IsAbs(statusFile) {
		return filepath.Clean(statusFile), nil
	}
	absolute, err := filepath.Abs(filepath.Join(cwd, statusFile))
	if err != nil {
		return "", fmt.Errorf("resolve status file path: %w", err)
	}
	return absolute, nil
}

func inferWorkdirFromStatusFile(statusFile, fallback string) string {
	clean := filepath.Clean(statusFile)
	marker := string(filepath.Separator) + "_bmad-output" + string(filepath.Separator)
	switch idx := strings.Index(clean, marker); {
	case idx > 0:
		return clean[:idx]
	case idx == 0:
		return string(filepath.Separator)
	default:
		return fallback
	}
}
