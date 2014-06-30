package system

import (
	"io"
	"lsf/panics"
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

type LogAccessMode string

var logAccess = struct{ Reader, Writer LogAccessMode }{"sys-log-reader", "sys-log-writer"}

// ~analogous to system.Document
type Log interface {
	Id() string
	Writer() LogWriter
	Reader() LogReader
}

type LogReader interface {
	// Read semantics for logs?
	ReadLine(n int64) (string, error)
	Tail() (string, error)
}

type LogWriter io.Writer // entertained

type syslog struct {
	id      string
	info    *os.FileInfo
	entries []string
	lock    Lock
}

func (sl *syslog) Id() string {
	panic("not impelmented")
}

func (sl *syslog) Tail() string {
	panic("not impelmented")
}

// this is NOT creating a log file. It is entirely analogous to document's newDocument.
func newLog(id string, fpath, fname string, data []string) (l *syslog, err error) {
	defer panics.Recover(&err)

	assertSystemObjectPath(fpath, fname) // panics

	panic("not implemented")
}
