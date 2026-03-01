package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/dlukt/bmad-autopilot/internal/orchestrator"
	"golang.org/x/term"
)

const (
	ctrlA = byte(1)
	ctrlC = byte(3)
	ctrlS = byte(19)
)

type stopHotkeySignal int

const (
	stopHotkeyRequest stopHotkeySignal = iota + 1
	stopHotkeyCancel
	stopHotkeyInterrupt
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

	previousState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, fmt.Errorf("enable raw terminal mode for stop hotkeys: %w", err)
	}

	signals := make(chan stopHotkeySignal, 4)
	done := make(chan struct{})

	go watchStopHotkeys(signals, done)
	go applyStopHotkeys(signals, done, stopController, cancel, output)

	fmt.Fprintln(output, "HOTKEYS: Ctrl+S requests graceful stop after the current command; Ctrl+A cancels it")

	return func() {
		close(done)
		_ = term.Restore(fd, previousState)
	}, nil
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
			}
		}
	}
}
