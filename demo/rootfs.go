package demo

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	_ "crypto/sha256" // register SHA256 hash for digest

	digest "github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	freeBSDDownload = "https://download.freebsd.org/ftp/releases/%s/%s/base.txz"
)

// DownloadRootfs downloads a FreeBSD root filesystem
func DownloadRootfs(arch, version string) (io.ReadCloser, int64, error) {
	req, err := http.Get(fmt.Sprintf(freeBSDDownload, arch, version))
	if err != nil {
		return nil, 0, err
	}
	if req.StatusCode < 200 || req.StatusCode > 299 {
		return nil, 0, fmt.Errorf("download: unexpected status %s (%d)", req.Status, req.StatusCode)
	}
	return req.Body, req.ContentLength, nil
}

// MakeImage constructs a single-layer FreeBSD OCI image from a given input tar
func MakeImage(rootfsFilename string, outputFilename string, arch string) error {
	tempDir, err := ioutil.TempDir("", "runj-demo-rootfs-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// setup directory structure
	imageDir := filepath.Join(tempDir, "image")
	blobDir := filepath.Join(imageDir, "blobs")
	err = os.MkdirAll(blobDir, 0755)
	if err != nil {
		return err
	}

	// extract
	fmt.Println("extracting...")
	err = unxz(rootfsFilename, filepath.Join(tempDir, "rootfs.tar"))
	if err != nil {
		return err
	}
	layerDigest, err := digestFile(filepath.Join(tempDir, "rootfs.tar"))
	if err != nil {
		return err
	}
	// re-compress
	fmt.Println("compressing...")
	layer, err := os.OpenFile(filepath.Join(tempDir, "rootfs.tar"), os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer layer.Close()
	compressedLayer, err := os.OpenFile(filepath.Join(tempDir, "rootfs.tar.gz"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer compressedLayer.Close()
	compressedWriter := gzip.NewWriter(compressedLayer)
	_, err = io.Copy(compressedWriter, layer)
	if err != nil {
		return err
	}
	compressedWriter.Close()
	compressedLayer.Sync()
	compressedInfo, err := compressedLayer.Stat()
	if err != nil {
		return err
	}
	fmt.Println("computing layer digest...")
	compressedDigest, err := digestFile(filepath.Join(tempDir, "rootfs.tar.gz"))
	if err != nil {
		return err
	}
	// blobs
	err = os.Rename(filepath.Join(tempDir, "rootfs.tar.gz"), filepath.Join(blobDir, compressedDigest.Encoded()))
	if err != nil {
		return err
	}
	// oci-layout
	err = ioutil.WriteFile(filepath.Join(imageDir, "oci-layout"), []byte(`{"imageLayoutVersion": "1.0.0"}`), 0644)
	if err != nil {
		return err
	}
	// config
	const freebsd = "freebsd"
	if arch == "" {
		arch = runtime.GOARCH
	}
	config := v1.Image{
		Architecture: arch,
		OS:           freebsd,
		RootFS: v1.RootFS{
			Type:    "layers",
			DiffIDs: []digest.Digest{layerDigest},
		},
		Author: "runj <runj@sbk.wtf>",
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}
	configDigest, err := writeBlob(blobDir, configJSON)
	if err != nil {
		return err
	}

	// manifest
	manifest := v1.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Config: v1.Descriptor{
			MediaType: v1.MediaTypeImageConfig,
			Digest:    configDigest,
			Size:      int64(len(configJSON)),
		},
		Layers: []v1.Descriptor{{
			MediaType: v1.MediaTypeImageLayerGzip,
			Digest:    compressedDigest,
			Size:      compressedInfo.Size(),
		}},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	manifestDigest, err := writeBlob(blobDir, manifestJSON)
	if err != nil {
		return err
	}
	// index
	index := v1.Index{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Manifests: []v1.Descriptor{{
			MediaType: v1.MediaTypeImageManifest,
			Digest:    manifestDigest,
			Size:      int64(len(manifestJSON)),
			Platform: &v1.Platform{
				Architecture: arch,
				OS:           freebsd,
			},
		}},
	}
	indexJSON, err := json.Marshal(index)
	if err != nil {
		return err
	}
	// image index
	err = ioutil.WriteFile(filepath.Join(imageDir, "index.json"), indexJSON, 0644)
	if err != nil {
		return err
	}

	// tar
	fmt.Println("tar...")
	return makeTar(imageDir, outputFilename)
}

func writeBlob(blobDir string, blob []byte) (digest.Digest, error) {
	d := digest.FromBytes(blob)
	fmt.Println("writing blob", string(d))
	err := ioutil.WriteFile(filepath.Join(blobDir, d.Encoded()), blob, 0644)
	if err != nil {
		return "", err
	}
	return d, nil
}

func unxz(in, out string) error {
	outFile, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		outFile.Close()
		if err != nil {
			os.Remove(out)
		}
	}()
	inFile, err := os.OpenFile(in, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer inFile.Close()
	xz := exec.Command("xz", "--decompress", "-")
	xz.Stdin = inFile
	xz.Stdout = outFile
	xz.Stderr = os.Stderr
	return xz.Run()
}

func digestFile(in string) (digest.Digest, error) {
	f, err := os.OpenFile(in, os.O_RDONLY, 0)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return digest.FromReader(f)
}

func makeTar(in string, out string) error {
	outFile, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		outFile.Close()
		if err != nil {
			os.Remove(out)
		}
	}()
	tw := tar.NewWriter(outFile)
	defer tw.Close()
	prefix := in + string(os.PathSeparator)
	return filepath.Walk(in, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		name := strings.TrimPrefix(path, prefix)
		hdr := &tar.Header{
			Name: name,
			Size: info.Size(),
			Mode: 0644,
		}
		err = tw.WriteHeader(hdr)
		if err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_RDONLY, 0)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}
