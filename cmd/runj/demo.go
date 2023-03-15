package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.sbk.wtf/runj/oci"

	"go.sbk.wtf/runj/runtimespec"

	"go.sbk.wtf/runj/demo"

	pb "github.com/cheggaaa/pb/v3"
	"github.com/spf13/cobra"
)

// demoCommand provides a subcommand for runj-specific demos.
// This command and its subcommands are not part of the OCI spec.
func demoCommand() *cobra.Command {
	demo := &cobra.Command{
		Use:   "demo",
		Short: "runj demos",
	}
	demo.AddCommand(downloadRootfsCommand())
	demo.AddCommand(specCommand())
	demo.AddCommand(imageCommand())
	return demo
}

func downloadRootfsCommand() *cobra.Command {
	dl := &cobra.Command{
		Use:   "download",
		Short: "download a FreeBSD rootfs",
		Long:  "Download the base.txz for a given FreeBSD release and architecture",
	}
	arch := dl.Flags().StringP("architecture", "a", "", "CPU architecture, like amd64")
	version := dl.Flags().StringP("version", "v", "", "FreeBSD version, like 12-RELEASE")
	outputFilename := dl.Flags().StringP("output", "o", "rootfs.txz", "Output filename")
	dl.RunE = func(cmd *cobra.Command, args []string) error {
		if *arch == "" {
			var err error
			*arch, err = demo.FreeBSDArch(dl.Context())
			if err != nil {
				return err
			}
			fmt.Println("Found arch: ", *arch)
		}
		if *version == "" {
			var err error
			*version, err = demo.FreeBSDVersion(dl.Context())
			if err != nil {
				return err
			}
			fmt.Println("Found version: ", *version)
		}
		f, err := os.OpenFile(*outputFilename, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		return downloadImage(*arch, *version, f)
	}
	return dl
}

func imageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oci-image [--input | --architecture <arch> --version <version>]",
		Short: "Create an OCI image",
		Long:  "Create an OCI image, optionally downloading if a rootfs file is not already present",
	}
	arch := cmd.Flags().StringP("architecture", "a", "", "CPU architecture, like amd64")
	version := cmd.Flags().StringP("version", "v", "", "FreeBSD version, like 12-RELEASE")
	inputFilename := cmd.Flags().StringP("input", "i", "", "Input rootfs (txz format)")
	outputFilename := cmd.Flags().StringP("output", "o", "image.tar", "Output filename")
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if (*arch == "" || *version == "") && *inputFilename == "" {
			return errors.New("missing required arguments; either provide --input or (--architecture and --version)")
		}
		if *inputFilename != "" && (*arch != "" || *version != "") {
			return errors.New("cannot provide --input and (--architecture or --version)")
		}
		return nil
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		disableUsage(cmd)
		if *inputFilename == "" {
			tempFile, err := os.CreateTemp("", "freebsd-image-")
			if err != nil {
				return err
			}
			defer func() {
				tempFile.Close()
				os.Remove(tempFile.Name())
			}()
			err = downloadImage(*arch, *version, tempFile)
			if err != nil {
				return err
			}
			*inputFilename = tempFile.Name()
		}
		fmt.Printf("Creating OCI image in file %s\n", *outputFilename)
		return demo.MakeImage(*inputFilename, *outputFilename, *arch)
	}
	return cmd
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

func specCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "generate an example config.json spec file",
		Long: `The spec command creates a new example config.json spec file for the bundle.

The spec generated is just a starter file. Editing of the spec is required to
achieve desired results. For example, the newly generated spec includes an args
parameter that is initially set to call the "sh" command when the container is
started.`,
	}
	bundlePath := cmd.Flags().StringP("bundle", "b", "", "Path to the bundle")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		spec := exampleSpec()
		if *bundlePath != "" {
			if err := os.Chdir(filepath.Clean(*bundlePath)); err != nil {
				return err
			}
		}
		if err := checkNoFile(oci.ConfigFileName); err != nil {
			return err
		}
		data, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(oci.ConfigFileName, data, 0666)
	}
	return cmd
}

func exampleSpec() *runtimespec.Spec {
	return &runtimespec.Spec{
		Version: runtimespec.Version,
		Process: &runtimespec.Process{
			Args: []string{"sh"},
		},
		Root: &runtimespec.Root{
			Path: "rootfs",
		},
		Mounts: []runtimespec.Mount{{
			Destination: "/dev",
			Source:      "devfs",
			Type:        "devfs",
			Options:     []string{"ruleset=4"},
		}},
		FreeBSD: &runtimespec.FreeBSD{
			Network: &runtimespec.FreeBSDNetwork{
				IPv4: &runtimespec.FreeBSDIPv4{
					Mode: runtimespec.FreeBSDIPv4ModeInherit,
				},
			},
		},
	}
}

func checkNoFile(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return fmt.Errorf("%s exists. Remove it first", path)
	}
	if !os.IsNotExist(err) {
		return err
	}
	return nil
}
