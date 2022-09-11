package util

import (
	"io"
	"os"
)

// CopyFile copies a file from source to dest
func CopyFile(source, dest string, perm os.FileMode) error {
	input, err := os.OpenFile(source, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer func() {
		output.Close()
		if err != nil {
			os.Remove(output.Name())
		}
	}()
	_, err = io.Copy(output, input)
	return err
}
