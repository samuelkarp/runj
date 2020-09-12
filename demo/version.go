package demo

import (
	"context"
	"os/exec"
	"strings"
)

func FreeBSDVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "freebsd-version")
	version, err := cmd.Output()
	if err != nil {
		return "", err
	}
	vers := strings.Split(string(version), "-")
	if len(vers) < 2 {
		return vers[0], nil
	}
	return strings.Join(vers[:2], "-"), nil
}

func FreeBSDArch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "uname", "-p")
	arch, err := cmd.Output()
	return strings.TrimSpace(string(arch)), err
}
