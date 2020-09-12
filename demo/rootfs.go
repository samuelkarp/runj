package demo

import (
	"fmt"
	"io"
	"net/http"
)

const (
	freeBSDDownload = "http://ftp.freebsd.org/pub/FreeBSD/releases/%s/%s/base.txz"
)

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
