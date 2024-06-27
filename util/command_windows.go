package util

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

func Command(name string, arg ...string) (string, error) {
	return CommandContext(context.Background(), name, arg...)
}

func CommandContext(ctx context.Context, name string, arg ...string) (string, error) {
	c := exec.CommandContext(ctx, name, arg...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%q: %w: %q", strings.Join(append([]string{name}, arg...), " "), err, out)
	}
	return strings.TrimSpace(string(out)), nil
}
