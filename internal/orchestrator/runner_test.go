package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/dlukt/bmad-autopilot/internal/brain"
)

func TestResolveStatusFilePath(t *testing.T) {
	cwd := filepath.FromSlash("/tmp/project")

	tests := []struct {
		name      string
		statusArg string
		want      string
	}{
		{
			name:      "default from cwd",
			statusArg: "",
			want:      filepath.Join(cwd, defaultStatusFile),
		},
		{
			name:      "relative path from cwd",
			statusArg: "custom/status.yaml",
			want:      filepath.Join(cwd, "custom/status.yaml"),
		},
		{
			name:      "absolute path untouched",
			statusArg: filepath.FromSlash("/opt/repo/_bmad-output/implementation-artifacts/sprint-status.yaml"),
			want:      filepath.FromSlash("/opt/repo/_bmad-output/implementation-artifacts/sprint-status.yaml"),
		},
	}

	for _, tt := range tests {
		got, err := resolveStatusFilePath(tt.statusArg, cwd)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.name, err)
		}
		if got != filepath.Clean(tt.want) {
			t.Fatalf("%s: expected %q, got %q", tt.name, filepath.Clean(tt.want), got)
		}
	}
}

func TestInferWorkdirFromStatusFile(t *testing.T) {
	fallback := filepath.FromSlash("/tmp/fallback")

	got := inferWorkdirFromStatusFile(
		filepath.FromSlash("/tmp/repo/_bmad-output/implementation-artifacts/sprint-status.yaml"),
		fallback,
	)
	if got != filepath.FromSlash("/tmp/repo") {
		t.Fatalf("expected /tmp/repo, got %q", got)
	}

	got = inferWorkdirFromStatusFile(
		filepath.FromSlash("/tmp/repo/custom/status.yaml"),
		fallback,
	)
	if got != fallback {
		t.Fatalf("expected fallback %q, got %q", fallback, got)
	}
}

type recordingExecutor struct {
	mu       sync.Mutex
	commands []string
	onRun    func()
}

func (e *recordingExecutor) Run(_ context.Context, action Action) (ExecResult, error) {
	e.mu.Lock()
	e.commands = append(e.commands, action.Command)
	e.mu.Unlock()

	if e.onRun != nil {
		e.onRun()
	}

	return ExecResult{}, nil
}

func (e *recordingExecutor) Commands() []string {
	e.mu.Lock()
	defer e.mu.Unlock()

	out := make([]string, len(e.commands))
	copy(out, e.commands)
	return out
}

func TestRunnerStopsAfterCurrentCommandWhenRequested(t *testing.T) {
	tempDir := t.TempDir()
	statusFile := filepath.Join(tempDir, "sprint-status.yaml")
	content := strings.TrimSpace(`
development_status:
  1-1-story-a: backlog
`) + "\n"
	if err := os.WriteFile(statusFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write sprint status: %v", err)
	}

	stopController := NewStopController()
	executor := &recordingExecutor{
		onRun: func() {
			stopController.RequestStop()
		},
	}

	runner := &Runner{
		cfg:      Config{StatusFile: statusFile},
		brain:    brain.DeterministicBrain{},
		executor: executor,
		stop:     stopController,
	}

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	commands := executor.Commands()
	if len(commands) != 1 {
		t.Fatalf("expected exactly one command before graceful stop, got %d (%v)", len(commands), commands)
	}
	if !strings.Contains(commands[0], "/bmad-bmm-create-story 1-1") {
		t.Fatalf("expected first command to be create-story for 1-1, got %q", commands[0])
	}
}
