package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/dlukt/bmad-autopilot/internal/orchestrator"
	"golang.org/x/term"
)

const (
	ctrlA = byte(1)
	ctrlC = byte(3)
	ctrlP = byte(16)
	ctrlS = byte(19)
)

type stopHotkeySignal int

const (
	stopHotkeyRequest stopHotkeySignal = iota + 1
	stopHotkeyCancel
	stopHotkeyInterrupt
	stopHotkeyTogglePoweroff
)

func startStopHotkeys(
	stopController *orchestrator.StopController,
	cancel context.CancelFunc,
	output io.Writer,
) (func(), error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return func() {}, nil
	}

	restoreTerminal, err := enableHotkeyTerminalMode()
	if err != nil {
		return nil, err
	}

	signals := make(chan stopHotkeySignal, 4)
	done := make(chan struct{})

	go watchStopHotkeys(signals, done)
	go applyStopHotkeys(signals, done, stopController, cancel, output)

	fmt.Fprintln(output, "HOTKEYS: Ctrl+S requests graceful stop after the current command; Ctrl+A cancels it")
	fmt.Fprintln(output, "HOTKEYS: Ctrl+P toggles poweroff at full completion")

	return func() {
		close(done)
		_ = restoreTerminal()
	}, nil
}

func enableHotkeyTerminalMode() (func() error, error) {
	savedState, err := readSttyState()
	if err != nil {
		return nil, fmt.Errorf("read terminal state: %w", err)
	}

	// Keep output processing enabled (no broken line wraps), but switch input
	// to single-byte mode and disable flow control so Ctrl+S is captured.
	if err := runStty("-icanon", "-echo", "-ixon", "-isig", "min", "1", "time", "0"); err != nil {
		return nil, fmt.Errorf("enable hotkey terminal mode: %w", err)
	}

	return func() error {
		return runStty(savedState)
	}, nil
}

func readSttyState() (string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	command := exec.Command("stty", "-g")
	command.Stdin = os.Stdin
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		errText := strings.TrimSpace(stderr.String())
		if errText == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, errText)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func runStty(args ...string) error {
	var stderr bytes.Buffer

	command := exec.Command("stty", args...)
	command.Stdin = os.Stdin
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		errText := strings.TrimSpace(stderr.String())
		if errText == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, errText)
	}

	return nil
}

func watchStopHotkeys(signals chan<- stopHotkeySignal, done <-chan struct{}) {
	buf := make([]byte, 1)
	for {
		select {
		case <-done:
			return
		default:
		}

		n, err := os.Stdin.Read(buf)
		if err != nil {
			return
		}
		if n == 0 {
			continue
		}

		switch buf[0] {
		case ctrlS:
			select {
			case signals <- stopHotkeyRequest:
			default:
			}
		case ctrlA:
			select {
			case signals <- stopHotkeyCancel:
			default:
			}
		case ctrlC:
			select {
			case signals <- stopHotkeyInterrupt:
			default:
			}
		case ctrlP:
			select {
			case signals <- stopHotkeyTogglePoweroff:
			default:
			}
		}
	}
}

func applyStopHotkeys(
	signals <-chan stopHotkeySignal,
	done <-chan struct{},
	stopController *orchestrator.StopController,
	cancel context.CancelFunc,
	output io.Writer,
) {
	for {
		select {
		case <-done:
			return
		case signal := <-signals:
			switch signal {
			case stopHotkeyRequest:
				stopController.RequestStop()
				fmt.Fprintln(output, "STOP: requested; current command will finish, then loop exits (Ctrl+A to continue)")
			case stopHotkeyCancel:
				stopController.CancelStop()
				fmt.Fprintln(output, "STOP: request canceled; loop will continue")
			case stopHotkeyInterrupt:
				fmt.Fprintln(output, "INTERRUPT: Ctrl+C pressed; canceling active command")
				cancel()
			case stopHotkeyTogglePoweroff:
				armed := stopController.TogglePoweroff()
				if armed {
					fmt.Fprintln(output, "POWEROFF: armed; system will power off when the full loop completes")
				} else {
					fmt.Fprintln(output, "POWEROFF: disarmed")
				}
			}
		}
	}
}
