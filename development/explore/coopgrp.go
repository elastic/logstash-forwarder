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
	"strconv"
	"time"
)

var options = struct {
	basepath     string
	pattern      string
	maxRecords   uint
	ageThreshold infoAge
	delaymsec    uint
	about        func() string
}{
	basepath:     ".",
	pattern:      "*",
	maxRecords:   100,
	ageThreshold: infoAge(time.Millisecond * 1000),
	delaymsec:    100,
}

func about() string {
	var s string = "explore/tracking module:\n"
	s += fmt.Sprintf("basepath:     %s\n", options.basepath)
	s += fmt.Sprintf("pattern:      %s\n", options.pattern)
	s += fmt.Sprintf("maxRecords:   %d\n", options.maxRecords)
	s += fmt.Sprintf("ageThreshold: %d\n", options.ageThreshold)
	s += fmt.Sprintf("delaymsec:    %d\n", options.delaymsec)
	return s
}
func init() {

	options.about = about

	flag.StringVar(&options.basepath, "p", options.basepath, "base path to track")
	flag.StringVar(&options.pattern, "n", options.pattern, "filename glob pattern")
	flag.UintVar(&options.maxRecords, "max-records", options.maxRecords, "maximum number of fs.Objects in cache")
	flag.UintVar(&options.delaymsec, "delay", options.delaymsec, "delay in msecs between reports")
	flag.Var(&options.ageThreshold, "age-limit", "limit on age of object in cache")

	flag.Usage = func() {
		log.Print(`
usage: <exe-name> [options]
options:
   -p:           path e.g. /var/log/webserver/
   -n:           pattern e.g. "apache2.log*"
   -delay:       msec wait before new report generation
   -age-limit:   max age of object in fs.Object cache
   -max-records: max number of objects in fs.Object cache
		`)
	}
	log.SetFlags(0)
}

type infoAge time.Duration

func (t *infoAge) String() string {
	return fmt.Sprintf("%d", *t)
}
func (t *infoAge) Set(vrep string) error {
	v, e := strconv.ParseInt(vrep, 10, 64)
	if e != nil {
		return e
	}
	var tt time.Duration = time.Duration(v) * time.Millisecond
	*t = infoAge(tt)
	return nil
}
func main() {

	defer panics.ExitHandler()

	flag.Parse()
	log.Println(about())
	opt := options
	var scout TrackScout = newTrackScout(opt.basepath, opt.pattern, uint16(opt.maxRecords), opt.ageThreshold)

	for {
		_, e := scout.Report()
		panics.OnError(e, "main", "scout.Report")

		time.Sleep(time.Millisecond * time.Duration(options.delaymsec))
	}

}
func nilInitializer() error { return nil }

type Component struct {
	initialize func() error
}
type TrackReport struct {
	Component
}
type TrackScout interface {
	Report() (*TrackReport, error)
}
type trackScout struct {
	Component
	options struct {
		maxRecords        uint16
		maxAge            time.Duration
		basepath, pattern string
	}
	objects *objcache
}

func newTrackScout(basepath, pattern string, maxRecords uint16, maxAge infoAge) TrackScout {
	ts := new(trackScout)
	ts.options.basepath = basepath
	ts.options.pattern = pattern
	ts.options.maxRecords = maxRecords
	ts.options.maxAge = time.Duration(maxAge)
	ts.initialize = ts.trackScoutInit
	return ts
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
		dir := path.Dir(gpattern)
		info, e := os.Stat(fspath)
		panics.OnError(e, "trackScout.trackScoutConst:", dir, fspath)
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
		if obj0, found := t.objects.cache[id]; found {
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
		t.objects.cache[id] = obj
		events[eventNum] = &capability.FileEvent{now, eventCode, obj}
		eventNum++
	}

	t.objects.Gc()

	for id, obj := range t.objects.cache {
		if yes, _ := t.objects.IsDeleted(id); !yes {
			if _, found := workingset[id]; !found {
				events = append(events, &capability.FileEvent{now, capability.TrackEvent.DeletedFile, obj})
				t.objects.MarkDeleted(id)
				log.Printf("marked deleted: %s %s", id, t.objects.cache[id])
			}
		}
	}

	for _, event := range events {
		if event.Code != capability.TrackEvent.KnownFile {
			log.Println(event)
		}
	}

	return nil, nil
}

func (c *Component) debugCompConst() error {
	log.Printf("Component.debugConst: comp-type: %T", c)
	c.initialize = nilInitializer
	return nil
}

func (t *trackScout) trackScoutInit() (err error) {
	defer panics.Recover(&err)

	t.objects = NewObjectCache(t.options.maxRecords, t.options.maxAge)
	t.initialize = nilInitializer

	return nil
}

type objcache struct {
	options struct {
		maxRecords uint16
		maxAge     time.Duration
	}
	cache map[string]fs.Object
}

func NewObjectCache(maxRecords uint16, maxAge time.Duration) *objcache {
	oc := new(objcache)
	oc.options.maxRecords = maxRecords
	oc.options.maxAge = maxAge
	oc.cache = make(map[string]fs.Object)
	return oc
}
func (oc *objcache) MarkDeleted(id string) bool {
	obj, found := oc.cache[id]
	if !found {
		return false
	}
	obj.SetFlags(1)
	return true
}
func (oc *objcache) IsDeleted(id string) (bool, error) {
	obj, found := oc.cache[id]
	if !found {
		return false, fmt.Errorf("no such object")
	}

	return obj.Flags() == uint8(1), nil
}
func (oc *objcache) IsDeleted0(id string) bool {
	obj, found := oc.cache[id]
	if !found {
		return true
	}

	return obj.Flags() == uint8(1)
}
func (oc *objcache) Gc() {
	n := 0
	if len(oc.cache) > int(oc.options.maxRecords) {
		for id, obj := range oc.cache {
			if obj.Age() > oc.options.maxAge && oc.IsDeleted0(id) {
				//				log.Printf("cache-gc: %s", obj)
				delete(oc.cache, id)
				n++
			}
		}
	}
	if n > 0 {
		log.Printf("GC: %d items removed", n)
		log.Printf("gc: %d %d", oc.options.maxRecords, len(oc.cache))
	}
}
