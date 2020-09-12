package main

import (
	"fmt"
	"io"
	"os"

	"sbk.wtf/runj/demo"

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
		defer f.Close()
		if err != nil {
			return err
		}
		fmt.Printf("Downloading image for %s %s into %s\n", *arch, *version, *outputFilename)
		rootfs, rootLen, err := demo.DownloadRootfs(*arch, *version)
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
	return dl
}
