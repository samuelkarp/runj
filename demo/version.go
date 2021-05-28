package demo

import (
	"context"
	"os/exec"
	"strings"
)

// FreeBSDVersion returns the current version as reported by freebsd-version(1)
func FreeBSDVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "freebsd-version")
	version, err := cmd.Output()
	if err != nil {
		return "", err
	}
	vers := strings.Split(strings.TrimSpace(string(version)), "-")
	if len(vers) < 2 {
		return vers[0], nil
	}
	return strings.Join(vers[:2], "-"), nil
}

// FreeBSDArch returns the current architecture as reported by uname(1)
func FreeBSDArch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "uname", "-p")
	arch, err := cmd.Output()
	return strings.TrimSpace(string(arch)), err
}
