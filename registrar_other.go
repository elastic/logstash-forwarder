// +build !windows

package main

import (
	"os"
)

func onRegistryWrite(path, tempfile string) error {
	if e := os.Rename(tempfile, path); e != nil {
		emit("registry rotate: rename of %s to %s - %s\n", tempfile, path, e)
		return e
	}
	return nil
}
