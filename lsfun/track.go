package lsfun

import (
	"fmt"
	"lsf"
	"lsf/fs"
	"lsf/panics"
	"os"
	"path"
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
	Events   []*FileEvent
}

func (t *TrackReport) String() string {
	return fmt.Sprintf("%020d %3d %s", t.Sequence, len(t.Events), t.Basepath)
}

type TrackScout interface {
	Report() (*TrackReport, error)
}

type trackScout struct {
	lsf.Component
	options struct {
		maxSize           uint16
		maxAge            fs.InfoAge
		basepath, pattern string
	}
	objects *fs.ObjectCache
}

func NewTrackScout(basepath, pattern string, maxSize uint16, maxAge fs.InfoAge) TrackScout {
	ts := new(trackScout)
	ts.options.basepath = basepath
	ts.options.pattern = pattern
	ts.options.maxSize = maxSize
	ts.options.maxAge = maxAge
	ts.Initialize = ts.trackScoutInit
	return ts
}

func (t *trackScout) trackScoutInit() (err error) {
	defer panics.Recover(&err)

	ageopt := t.options.maxAge != fs.InfoAge(0)
	sizeopt := t.options.maxSize != uint16(0)
	switch {
	case ageopt && sizeopt:
		panic("trackScout.trackScoutInit: only one of age or size limits can be specified for the tracking scout object cache")
	case ageopt:
		t.objects = fs.NewTimeWindowObjectCache(t.options.maxAge)
	case sizeopt:
		t.objects = fs.NewFixedSizeObjectCache(t.options.maxSize)
	default:
		panic("trackScout.trackScoutInit: one of age or size limits must be specified for the tracking scout object cache")
	}
	t.Initialize = lsf.NilInitializer

	return nil
}

func (t *trackScout) Report() (report *TrackReport, err error) {
//	panics := panics.ForFunc("trackScout.Report")
	defer panics.Recover(&err)

	e := t.Initialize()
	panics.OnError(e, "trackScout.Report:", "initialize:")

	gpattern := path.Join(t.options.basepath, t.options.pattern)
	now := time.Now()

	fspaths, e := fs.FindMatchingPaths(t.options.basepath, t.options.pattern)
	panics.OnError(e, "trackScout.Report:", "filepath.Glob", gpattern)

	workingset := make(map[string]fs.Object)
	for _, fspath := range fspaths {
		// REVU: resolve this issue of relative paths. It is a pain and design smell.
		_ = path.Dir(gpattern)

		info, e := os.Stat(fspath)
		if e != nil {
			// ignore: os provided both file names and Stat func.
			// A brief flicker of fs life.
			continue
		}
		if info.IsDir() {
			continue
		}
		fsobj := fs.AsObjectAt(info, now)
		workingset[fsobj.Id()] = fsobj
	}

	var events []*FileEvent = make([]*FileEvent, len(workingset))
	var eventCode FileEventCode
	var eventNum int

	// REVU: if polling period is longer than rollover period
	//       then MOD events will be missed in event stream.
	for id, obj := range workingset {
		if obj0, found := t.objects.Cache[id]; found {
			if fs.Renamed0(obj, obj0) {
				eventCode = TrackEvent.RenamedFile
			} else if fs.Modified0(obj, obj0) {
				eventCode = TrackEvent.ModifiedFile
			} else {
				eventCode = TrackEvent.KnownFile
			}
		} else {
			eventCode = TrackEvent.NewFile
		}
		t.objects.Cache[id] = obj
		events[eventNum] = &FileEvent{now, eventCode, obj}
		eventNum++
	}

	t.objects.Gc()

	for id, obj := range t.objects.Cache {
		if yes, _ := t.objects.IsDeleted(id); !yes {
			if _, found := workingset[id]; !found {
				// use timestamp of original fs.Object
				events = append(events, &FileEvent{now, TrackEvent.DeletedFile, obj})
				t.objects.MarkDeleted(id)
			}
		}
	}

	report = &TrackReport{uint64(0), t.options.basepath, events}

	return report, nil
}
