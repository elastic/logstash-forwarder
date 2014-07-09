// Licensed to Elasticsearch under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package system

import (
	"github.com/elasticsearch/kriterium/panics"
	"io"
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
