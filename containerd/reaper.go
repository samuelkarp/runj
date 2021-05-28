package containerd

import (
	"context"
	"os"
	"os/signal"

	"github.com/containerd/containerd/sys/reaper"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// Consider contributing the following to upstream containerd

// SetReaper sets this process as the reaper for its orphaned descendant processes
func SetReaper() error {
	return procReapAcquire()
}

const (
	//nolint:golint // match symbol name
	_P_PID = 0 // https://github.com/freebsd/freebsd-src/blob/098dbd7ff7f3da9dda03802cdb2d8755f816eada/sys/sys/wait.h#L109
	//nolint:golint // match symbol name
	_PROC_REAP_ACQUIRE = 2 // https://github.com/freebsd/freebsd-src/blob/098dbd7ff7f3da9dda03802cdb2d8755f816eada/sys/sys/procctl.h#L49
)

func procReapAcquire() error {
	pid := unix.Getpid()
	_, _, err := unix.Syscall(unix.SYS_PROCCTL, _P_PID, uintptr(pid), _PROC_REAP_ACQUIRE)
	return err
}

// End section for upstream contribution

// SetupReaperSignals configures the current process as a reaper, sets up a
// signal handler to receive SIGCHLD,and processes SIGCHLD events with
// containerd's reaper package.
func SetupReaperSignals(ctx context.Context, logger *logrus.Entry) error {
	if err := SetReaper(); err != nil {
		return err
	}

	signals := make(chan os.Signal, 32)
	signal.Notify(signals, unix.SIGCHLD)
	go handleSignals(ctx, logger, signals)
	return nil
}

// This function was copied from containerd
// https://github.com/containerd/containerd/blob/d8208e2e376ed972a111de69e98e1661bb7224e9/runtime/v2/shim/shim_unix.go#L73-L90
// Retrieved March 11, 2021
func handleSignals(ctx context.Context, logger *logrus.Entry, signals chan os.Signal) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case s := <-signals:
			switch s {
			case unix.SIGCHLD:
				if err := reaper.Reap(); err != nil {
					logger.WithError(err).Error("reap exit status")
				}
			case unix.SIGPIPE:
			}
		}
	}
}
