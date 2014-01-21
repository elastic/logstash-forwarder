// +build !windows

package main

import (
	"os"
)

func SharedOpen(path string) (*os.File, error) {
	return os.Open(path)
}

