package system

import (
	"io"
	"os"
	"github.com/elasticsearch/kriterium/panics"
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

var LogAccess = struct{ Reader, Writer LogAccessMode }{"sys-log-reader", "sys-log-writer"}

// ~analogous to system.Document, Log is a system FS object
type Log interface {
	Id() string
	Writer() LogWriter
	Reader() LogReader
}

// REVU: .. ? .. TODO: REVU:
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
// REVU: TODO: change both to newTTTObject
func newLog(id string, fpath, fname string, data []string) (l *syslog, err error) {
	defer panics.Recover(&err)

	assertSystemObjectPath(fpath, fname) // panics

	panic("not implemented")
}
