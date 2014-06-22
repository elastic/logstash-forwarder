package system

import (
	"os"
)

/*
 * A System Log is a log structured, append-only, fixed-width, ordered list
 * of system records such as FileEvents, etc. Like System Documents (which are
 * k/v objects), these have semantics of array rings.
 */

// ----------------------------------------------------------------------------
// Log
// ----------------------------------------------------------------------------

type LogId string

type Log interface {
	//	io.Writer
	Id() LogId
	Tail() string
}

type syslog struct {
	id      LogId
	info    *os.FileInfo
	entries []string
	lock    Lock
}
