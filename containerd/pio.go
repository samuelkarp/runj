package containerd

import (
	"context"
	"io"
	"os"
	"syscall"

	"github.com/containerd/fifo"
)

type stdio struct {
	stdin  io.ReadWriteCloser
	stdout io.ReadWriteCloser
	stderr io.ReadWriteCloser
}

func setupIO(ctx context.Context, stdin, stdout, stderr string) (stdio, error) {
	io := stdio{}
	if _, err := os.Stat(stdin); err == nil {
		io.stdin, err = fifo.OpenFifo(ctx, stdin, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return io, err
		}
	}
	if _, err := os.Stat(stdout); err == nil {
		io.stdout, err = fifo.OpenFifo(ctx, stdout, syscall.O_WRONLY, 0)
		if err != nil {
			return io, err
		}
	}
	if _, err := os.Stat(stderr); err == nil {
		io.stderr, err = fifo.OpenFifo(ctx, stderr, syscall.O_WRONLY, 0)
		if err != nil {
			return io, err
		}
	}
	return io, nil
}

func (s stdio) Close() error {
	if s.stdin != nil {
		s.stdin.Close()
	}
	if s.stdout != nil {
		s.stdout.Close()
	}
	if s.stderr != nil {
		s.stderr.Close()
	}
	return nil
}
