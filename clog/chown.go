//go:build !linux

package clog

import (
	"os"
)

func chown(_ string, _ os.FileInfo) error {
	return nil
}
