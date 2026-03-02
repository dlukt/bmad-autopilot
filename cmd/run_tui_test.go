package cmd

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dlukt/bmad-autopilot/internal/orchestrator"
)

func TestTUIOutputWriterDoesNotBlockWhenChannelIsFull(t *testing.T) {
	logs := make(chan string, 1)
	logs <- "occupied"

	writer := &tuiOutputWriter{logs: logs}
	done := make(chan struct{})

	go func() {
		_, _ = writer.Write([]byte("hello"))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("write blocked while log channel was full")
	}
}

func TestTUIOutputWriterEmitsDropSummaryOnRecovery(t *testing.T) {
	logs := make(chan string, 1)
	logs <- "occupied"

	writer := &tuiOutputWriter{logs: logs}
	_, _ = writer.Write([]byte("dropped-first"))

	<-logs

	_, _ = writer.Write([]byte("next"))
	msg := <-logs

	if !strings.Contains(msg, "TUI: output backpressure; dropped 1 chunk(s)") {
		t.Fatalf("expected drop summary, got %q", msg)
	}
	if !strings.Contains(msg, "next") {
		t.Fatalf("expected next chunk payload, got %q", msg)
	}
}

func TestRunTUIFinalizesOnlyAfterLogChannelClosed(t *testing.T) {
	model := newRunTUIModel(
		make(chan string),
		func() (orchestrator.RunOutcome, error) { return orchestrator.RunOutcomeCompleted, nil },
		orchestrator.NewStopController(),
		func() {},
	)

	updated, cmd := model.Update(tuiRunFinishedMsg{
		outcome: orchestrator.RunOutcomeCompleted,
		err:     nil,
	})
	model = updated.(runTUIModel)
	if model.done {
		t.Fatal("expected run to remain unfinished until log channel is closed")
	}
	if cmd != nil {
		t.Fatal("expected no quit/finalization command before log channel closure")
	}

	updated, cmd = model.Update(tuiLogMsg("tail output\n"))
	model = updated.(runTUIModel)
	if model.done {
		t.Fatal("expected run to remain unfinished while pending logs are still being consumed")
	}
	if cmd == nil {
		t.Fatal("expected follow-up wait command while log stream is open")
	}

	updated, cmd = model.Update(tuiLogClosedMsg{})
	model = updated.(runTUIModel)
	if !model.done {
		t.Fatal("expected run to finalize once log channel is closed")
	}
	if cmd != nil {
		t.Fatal("expected no auto-quit command when poweroff is not armed")
	}
	if !strings.Contains(model.logBuffer.String(), "tail output") {
		t.Fatalf("expected tail output to be preserved, got %q", model.logBuffer.String())
	}
}

func TestRunTUIPoweroffQuitWaitsForLogChannelClose(t *testing.T) {
	stopController := orchestrator.NewStopController()
	stopController.TogglePoweroff()

	model := newRunTUIModel(
		make(chan string),
		func() (orchestrator.RunOutcome, error) { return orchestrator.RunOutcomeCompleted, nil },
		stopController,
		func() {},
	)

	updated, cmd := model.Update(tuiRunFinishedMsg{
		outcome: orchestrator.RunOutcomeCompleted,
		err:     nil,
	})
	model = updated.(runTUIModel)
	if cmd != nil {
		t.Fatal("expected no quit command before log channel closure")
	}

	updated, cmd = model.Update(tuiLogClosedMsg{})
	model = updated.(runTUIModel)
	if !model.done {
		t.Fatal("expected run to finalize when logs close")
	}
	if cmd == nil {
		t.Fatal("expected quit command after logs close for armed poweroff")
	}

	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("expected tea.Quit command after logs close")
	}
}
