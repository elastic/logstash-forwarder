package lsfun

import (
	"fmt"
	"lsf/fs"
	"os"
	"time"
)

// ----------------------------------------------------------------------
// tracked file event
// ----------------------------------------------------------------------

type FileEventCode string

func (t FileEventCode) String() string { return string(t) }

// enum
var TrackEvent = struct {
	NewFile, KnownFile, ModifiedFile, DeletedFile, RenamedFile FileEventCode
}{
	NewFile:      "TRK",
	KnownFile:    "NOP",
	RenamedFile:  "NAM",
	ModifiedFile: "MOD",
	DeletedFile:  "DEL",
}

type FileEvent struct {
	Timestamp time.Time
	Code      FileEventCode
	File      fs.Object
}

func (t *FileEvent) String() string {
//	return fmt.Sprintf("%d %3s oid:%s %s", t.Timestamp.UnixNano(), t.Code.String(), t.File.Id(), fileStatString(t.File.Info()))
	return fmt.Sprintf("%d %3s %s", t.Timestamp.UnixNano(), t.Code.String(), t.File)
}

func fileStatString(f os.FileInfo) string {
	if f == nil {
		return "BUG - nil"
	}
	return fmt.Sprintf("%020d %s %012d %s", f.Size(), f.Mode(), f.ModTime().Unix(), f.Name())
}

// ----------------------------------------------------------------------
// tracker report
// ----------------------------------------------------------------------

// assumes track focuses on a specific (base)path
type TrackReport struct {
	Sequence uint64
	Basepath string
	Events   []FileEvent
}

func (t *TrackReport) String() string {
	return fmt.Sprintf("%020d %3d %s", t.Sequence, len(t.Events), t.Basepath)
}

// function () OID (info os.FileInfo) FSObject {}
// function Track

// function TrackAnalysis -> TrackReport

//
