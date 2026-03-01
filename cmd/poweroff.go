package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func runSystemPoweroff(ctx context.Context, stdout io.Writer, stderr io.Writer) error {
	command := exec.CommandContext(ctx, "sudo", "systemctl", "poweroff")
	command.Stdin = os.Stdin
	command.Stdout = stdout
	command.Stderr = stderr

	if err := command.Run(); err != nil {
		return fmt.Errorf("poweroff command failed: %w", err)
	}
	return nil
}
