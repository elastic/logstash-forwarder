package main

import (
	"flag"
	"fmt"
	"log"
	"lsf/capability"
	"lsf/fs"
	"lsf/panics"
	"os"
	"path"
	"time"
)

var options = struct {
	basepath  string
	pattern   string
	maxSize   uint
	maxAge    fs.InfoAge
	delaymsec uint
	about     func() string
}{
	basepath:  ".",
	pattern:   "*",
	maxSize:   0,
	maxAge:    fs.InfoAge(0),
	delaymsec: 100,
}

func about() string {
	var s string = "explore/tracking module:\n"
	s += fmt.Sprintf("basepath:  %s\n", options.basepath)
	s += fmt.Sprintf("pattern:   %s\n", options.pattern)
	s += fmt.Sprintf("maxSize:   %d\n", options.maxSize)
	s += fmt.Sprintf("maxAge:    %d\n", options.maxAge)
	s += fmt.Sprintf("delaymsec: %d\n", options.delaymsec)
	return s
}
func init() {

	options.about = about

	flag.StringVar(&options.basepath, "p", options.basepath, "base path to track")
	flag.StringVar(&options.pattern, "n", options.pattern, "filename glob pattern")
	flag.UintVar(&options.delaymsec, "delay", options.delaymsec, "delay in msecs between reports")
	flag.UintVar(&options.maxSize, "max-size", options.maxSize, "maximum number of fs.Objects in cache")
	flag.Var(&options.maxAge, "max-age", "limit on age of object in cache")

	flag.Usage = func() {
		log.Print(`
usage: <exe-name> [options]
options:
   -p:           path e.g. /var/log/webserver/
   -n:           pattern e.g. "apache2.log*"
   -delay:       msec wait before new report generation
   -age-limit:   max age of object in fs.Object cache. mutually exlusive w/ -max-records
   -max-records: max number of objects in fs.Object cache. mutually exlusive w/ -age-limit
		`)
	}
	log.SetFlags(0)
}

//panics
func validateGcOptions() {
	ageopt := options.maxAge != fs.InfoAge(0)
	sizeopt := options.maxSize != uint(0)
	if ageopt && sizeopt {
		panic("only one of age or size limits can be specified for the cache")
	} else if !(ageopt || sizeopt) {
		panic("one of age or size limits must be specified for the cache")
	}
}
func main() {

	defer panics.ExitHandler()

	flag.Parse()
	validateGcOptions()
	log.Println(about())

	opt := options
	var scout TrackScout = newTrackScout(opt.basepath, opt.pattern, uint16(opt.maxSize), opt.maxAge)

	for {
		_, e := scout.Report()
		panics.OnError(e, "main", "scout.Report")

		time.Sleep(time.Millisecond * time.Duration(options.delaymsec))
	}

}

// --- todo: extract lsf/component

func nilInitializer() error { return nil }

type Component struct {
	initialize func() error
}

func (c *Component) debugCompConst() error {
	log.Printf("Component.debugConst: comp-type: %T", c)
	c.initialize = nilInitializer
	return nil
}

// --- todo: extract lsf/capability

type TrackReport struct {
	Component
}

// --- todo: extract lsf/capability

type TrackScout interface {
	Report() (*TrackReport, error)
}
type trackScout struct {
	Component
	options struct {
		maxSize           uint16
		maxAge            fs.InfoAge
		basepath, pattern string
	}
	objects *fs.ObjectCache
}

func newTrackScout(basepath, pattern string, maxSize uint16, maxAge fs.InfoAge) TrackScout {
	ts := new(trackScout)
	ts.options.basepath = basepath
	ts.options.pattern = pattern
	ts.options.maxSize = maxSize
	ts.options.maxAge = maxAge
	ts.initialize = ts.trackScoutInit
	return ts
}

func (t *trackScout) trackScoutInit() (err error) {
	defer panics.Recover(&err)

	ageopt := t.options.maxAge != fs.InfoAge(0)
	sizeopt := t.options.maxSize != uint16(0)
	switch {
	case ageopt && sizeopt:
		panic("only one of age or size limits can be specified for the tracking scout object cache")
	case ageopt:
		t.objects = fs.NewTimeWindowObjectCache(t.options.maxAge)
	case sizeopt:
		t.objects = fs.NewFixedSizeObjectCache(t.options.maxSize)
	default:
		panic("one of age or size limits must be specified for the tracking scout object cache")
	}
	t.initialize = nilInitializer

	return nil
}

func (t *trackScout) Report() (report *TrackReport, err error) {
	panics := panics.ForFunc("Report")
	defer panics.Recover(&err)

	e := t.initialize()
	panics.OnError(e, "trackScout.Report:", "initialize:")

	gpattern := path.Join(t.options.basepath, t.options.pattern)
	now := time.Now()

	fspaths, e := fs.FindMatchingPaths(t.options.basepath, t.options.pattern)
	panics.OnError(e, "trackScout.trackScoutConst:", "filepath.Glob", gpattern)

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

	var events []*capability.FileEvent = make([]*capability.FileEvent, len(workingset))
	var eventCode capability.FileEventCode
	var eventNum int

	// REVU: if polling period is longer than rollover period
	//       then MOD events will be missed in event stream.
	for id, obj := range workingset {
		if obj0, found := t.objects.Cache[id]; found {
			if fs.Renamed0(obj, obj0) {
				eventCode = capability.TrackEvent.RenamedFile
			} else if fs.Modified0(obj, obj0) {
				eventCode = capability.TrackEvent.ModifiedFile
			} else {
				eventCode = capability.TrackEvent.KnownFile
			}
		} else {
			eventCode = capability.TrackEvent.NewFile
		}
		t.objects.Cache[id] = obj
		events[eventNum] = &capability.FileEvent{now, eventCode, obj}
		eventNum++
	}

	t.objects.Gc()

	for id, obj := range t.objects.Cache {
		if yes, _ := t.objects.IsDeleted(id); !yes {
			if _, found := workingset[id]; !found {
				// use timestamp of original fs.Object
				events = append(events, &capability.FileEvent{now, capability.TrackEvent.DeletedFile, obj})
				t.objects.MarkDeleted(id)
				//				log.Printf("marked deleted: %s %s", id, t.objects.Cache[id])
			}
		}
	}

	for _, event := range events {
		if event.Code != capability.TrackEvent.KnownFile { // printing NOP events gets noisy
			log.Println(event)
		}
	}

	return nil, nil
}
