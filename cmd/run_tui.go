package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dlukt/bmad-autopilot/internal/orchestrator"
	"github.com/spf13/cobra"
)

type tuiLogMsg string
type tuiLogClosedMsg struct{}

type tuiRunFinishedMsg struct {
	outcome orchestrator.RunOutcome
	err     error
}

type tuiOutputWriter struct {
	logs chan<- string
	mu   sync.Mutex

	droppedChunks int
	droppedBytes  int
}

func (w *tuiOutputWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	chunk := string(p)

	w.mu.Lock()
	if w.droppedChunks > 0 {
		chunk = fmt.Sprintf(
			"TUI: output backpressure; dropped %d chunk(s) (%d bytes)\n%s",
			w.droppedChunks,
			w.droppedBytes,
			chunk,
		)
	}

	if w.tryEnqueue(chunk) {
		w.droppedChunks = 0
		w.droppedBytes = 0
		w.mu.Unlock()
		return len(p), nil
	}

	w.droppedChunks++
	w.droppedBytes += len(p)
	w.mu.Unlock()

	return len(p), nil
}

func (w *tuiOutputWriter) tryEnqueue(chunk string) bool {
	select {
	case w.logs <- chunk:
		return true
	default:
		return false
	}
}

type runTUIModel struct {
	viewport viewport.Model
	ready    bool
	width    int
	height   int

	statusStyle lipgloss.Style
	logBuffer   *bytes.Buffer

	logs           chan string
	run            func() (orchestrator.RunOutcome, error)
	stopController *orchestrator.StopController
	cancel         context.CancelFunc

	runFinished bool
	logsClosed  bool
	done        bool
	runOutcome  orchestrator.RunOutcome
	runErr      error
}

func newRunTUIModel(
	logs chan string,
	run func() (orchestrator.RunOutcome, error),
	stopController *orchestrator.StopController,
	cancel context.CancelFunc,
) runTUIModel {
	return runTUIModel{
		logs:           logs,
		run:            run,
		stopController: stopController,
		cancel:         cancel,
		logBuffer:      &bytes.Buffer{},
		statusStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("63")).
			Padding(0, 1),
	}
}

func (m runTUIModel) Init() tea.Cmd {
	return tea.Batch(waitForTUILog(m.logs), runTUILoop(m.run, m.logs))
}

func waitForTUILog(logs <-chan string) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-logs
		if !ok {
			return tuiLogClosedMsg{}
		}
		return tuiLogMsg(chunk)
	}
}

func runTUILoop(run func() (orchestrator.RunOutcome, error), logs chan string) tea.Cmd {
	return func() tea.Msg {
		outcome, err := run()
		close(logs)
		return tuiRunFinishedMsg{
			outcome: outcome,
			err:     err,
		}
	}
}

func (m runTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		contentHeight := msg.Height - 1
		if contentHeight < 1 {
			contentHeight = 1
		}

		if !m.ready {
			m.viewport = viewport.New(msg.Width, contentHeight)
			m.viewport.SetContent(m.logBuffer.String())
			m.viewport.GotoBottom()
			m.ready = true
			return m, nil
		}

		m.viewport.Width = msg.Width
		m.viewport.Height = contentHeight
		return m, nil

	case tuiLogMsg:
		m.appendLog(string(msg))
		return m, waitForTUILog(m.logs)

	case tuiLogClosedMsg:
		m.logsClosed = true
		return m, m.finalizeRunIfReady()

	case tuiRunFinishedMsg:
		m.runFinished = true
		m.runOutcome = msg.outcome
		m.runErr = msg.err
		return m, m.finalizeRunIfReady()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			if !m.done {
				m.stopController.RequestStop()
				m.appendSystemLine("STOP: requested; current command will finish, then loop exits (Ctrl+A to continue)")
			}
			return m, nil

		case "ctrl+a":
			if !m.done {
				m.stopController.CancelStop()
				m.appendSystemLine("STOP: request canceled; loop will continue")
			}
			return m, nil

		case "ctrl+p":
			if !m.done {
				armed := m.stopController.TogglePoweroff()
				if armed {
					m.appendSystemLine("POWEROFF: armed; system will power off when the full loop completes")
				} else {
					m.appendSystemLine("POWEROFF: disarmed")
				}
			}
			return m, nil

		case "ctrl+c":
			if !m.done {
				m.cancel()
				m.appendSystemLine("INTERRUPT: Ctrl+C pressed; canceling active command")
				return m, nil
			}
			return m, tea.Quit

		case "q", "enter":
			if m.done {
				return m, tea.Quit
			}
			return m, nil
		}
	}

	if !m.ready {
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m runTUIModel) View() string {
	if !m.ready {
		return "Initializing TUI..."
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.viewport.View(), m.statusBar())
}

func (m *runTUIModel) appendSystemLine(line string) {
	m.appendLog(line + "\n")
}

func (m *runTUIModel) appendLog(chunk string) {
	if chunk == "" {
		return
	}

	follow := true
	if m.ready {
		follow = m.viewport.AtBottom()
	}

	_, _ = io.WriteString(m.logBuffer, chunk)
	if !m.ready {
		return
	}
	m.viewport.SetContent(m.logBuffer.String())
	if follow {
		m.viewport.GotoBottom()
	}
}

func (m runTUIModel) statusBar() string {
	state := "running"
	switch {
	case !m.done && m.runFinished:
		state = "flushing"
	case !m.done:
		state = "running"
	case m.runErr != nil:
		state = "error"
	case m.runOutcome == orchestrator.RunOutcomeCompleted:
		state = "completed"
	case m.runOutcome == orchestrator.RunOutcomeStopped:
		state = "stopped"
	default:
		state = "done"
	}

	stopState := "off"
	if m.stopController.ShouldStop() {
		stopState = "on"
	}
	poweroffState := "off"
	if m.stopController.PoweroffArmed() {
		poweroffState = "on"
	}

	controls := "Ctrl+S stop | Ctrl+A resume | Ctrl+P poweroff | Ctrl+C cancel | q quit"
	if !m.done {
		controls = "Ctrl+S stop | Ctrl+A resume | Ctrl+P poweroff | Ctrl+C cancel | Up/Down scroll"
	}

	text := fmt.Sprintf("State: %s | Stop: %s | Poweroff: %s | %s", state, stopState, poweroffState, controls)
	if m.width > 0 {
		text = truncateText(text, m.width-2)
		return m.statusStyle.Width(m.width).Render(text)
	}
	return m.statusStyle.Render(text)
}

func (m *runTUIModel) finalizeRunIfReady() tea.Cmd {
	if m.done || !m.runFinished || !m.logsClosed {
		return nil
	}

	m.done = true

	if m.runErr != nil {
		m.appendSystemLine(fmt.Sprintf("ERROR: %v", m.runErr))
	} else {
		switch m.runOutcome {
		case orchestrator.RunOutcomeCompleted:
			m.appendSystemLine("DONE: run completed")
		case orchestrator.RunOutcomeStopped:
			m.appendSystemLine("DONE: run stopped gracefully")
		default:
			m.appendSystemLine("DONE: run ended")
		}
	}

	if m.runErr == nil && m.runOutcome == orchestrator.RunOutcomeCompleted && m.stopController.PoweroffArmed() {
		m.appendSystemLine("POWEROFF: armed and loop completed; exiting TUI to run `sudo systemctl poweroff`")
		return tea.Quit
	}

	m.appendSystemLine("Press q to exit")
	return nil
}

func truncateText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxWidth {
		return text
	}
	if maxWidth <= 3 {
		return string(runes[:maxWidth])
	}
	return string(runes[:maxWidth-3]) + "..."
}

func runWithTUI(cmd *cobra.Command, opts *rootOptions) error {
	runCtx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	stopController := orchestrator.NewStopController()
	logs := make(chan string, 512)
	writer := &tuiOutputWriter{logs: logs}

	run := func() (orchestrator.RunOutcome, error) {
		return runOrchestrator(runCtx, opts, stopController, writer)
	}

	model := newRunTUIModel(logs, run, stopController, cancel)
	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return err
	}

	final, ok := finalModel.(runTUIModel)
	if !ok {
		return nil
	}
	if final.runErr != nil {
		return final.runErr
	}

	if final.runOutcome == orchestrator.RunOutcomeCompleted && stopController.PoweroffArmed() {
		fmt.Fprintln(cmd.OutOrStdout(), "POWEROFF: armed and loop completed; running `sudo systemctl poweroff`")
		if err := runSystemPoweroff(runCtx, cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
			return err
		}
	}

	return nil
}
