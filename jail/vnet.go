package jail

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
)

type VNetMove bool

const (
	VNetMoveIn  VNetMove = false
	VNetMoveOut          = true
)

const ifconfig = "/sbin/ifconfig" // _PATH_IFCONFIG from "/include/paths.h"

func MoveVNetInterfaces(ctx context.Context, ociConfig *runtimespec.Spec, j Jail, reverse VNetMove) error {
	if ociConfig == nil || ociConfig.FreeBSD == nil || ociConfig.FreeBSD.Jail == nil {
		return nil
	}
	if len(ociConfig.FreeBSD.Jail.VnetInterfaces) == 0 {
		return nil
	}
	if j.id() == 0 {
		return errors.New("cannot move vnet interface to jail 0")
	}
	vnetArg := "vnet"
	if reverse {
		vnetArg = "-vnet"
	}
	for _, iface := range ociConfig.FreeBSD.Jail.VnetInterfaces {
		cmd := exec.CommandContext(ctx, filepath.Clean(ifconfig), iface, vnetArg, j.id().String())
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("ifconfig: %q: %w", out, err)
		}
	}
	return nil
}
