//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cheggaaa/pb/v3"

	"go.sbk.wtf/runj/demo"
)

const (
	cacheFormat = "%s.%s.base.txz"
)

var (
	baseCache  string
	fullRootfs string
)

func TestMain(m *testing.M) {
	baseCache = filepath.Join(os.TempDir(), "runj-integ-test-cache", "base")
	fullRootfs = filepath.Join(os.TempDir(), "runj-integ-test-cache", "rootfs")
	if err := prepareRootfs(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to prepare rootfs: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	if err := cleanup(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to remove rootfs: %v\n", err)
	}
	os.Exit(code)
}

func cleanup() error {
	fmt.Println("Cleaning rootfs...")
	if out, err := exec.Command("chflags", "-R", "noschg", fullRootfs).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, string(out))
		return err
	}
	return os.RemoveAll(fullRootfs)
}

func prepareRootfs() error {
	if _, err := os.Stat(fullRootfs); err == nil {
		return fmt.Errorf("prepare: %q must not exist", fullRootfs)
	}
	if err := os.MkdirAll(baseCache, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(fullRootfs, 0o755); err != nil {
		return err
	}

	arch, err := demo.FreeBSDArch(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("Found arch: ", arch)

	version, err := demo.FreeBSDVersion(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("Found version: ", version)

	// check if the file is already downloaded
	cacheFile := filepath.Join(baseCache, fmt.Sprintf(cacheFormat, version, arch))
	if _, err := os.Stat(cacheFile); err != nil {
		f, err := os.OpenFile(cacheFile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		err = downloadImage(arch, version, f)
		f.Close()
		if err != nil {
			return err
		}
	}
	// extract
	fmt.Println("Extracting rootfs...")
	out, err := exec.Command("tar", "--directory", fullRootfs, "-xJf", cacheFile).CombinedOutput()
	fmt.Println(string(out))
	return err
}

func downloadImage(arch, version string, f *os.File) error {
	fmt.Printf("Downloading image for %s %s into %s\n", arch, version, f.Name())
	rootfs, rootLen, err := demo.DownloadRootfs(arch, version)
	if err != nil {
		return err
	}
	defer rootfs.Close()
	bar := pb.Full.Start64(rootLen)
	barReader := bar.NewProxyReader(rootfs)
	_, err = io.Copy(f, barReader)
	bar.Finish()
	return err
}
