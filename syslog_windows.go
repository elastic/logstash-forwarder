package main

import (
	"errors"
	"io"
)

func configureSyslog() (io.Writer, error) {
	return nil, errors.New("Logging to syslog not supported on this platform")
}
