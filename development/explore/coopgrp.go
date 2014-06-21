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
	maxRecords:   0,
	ageThreshold: infoAge(0),
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
   -age-limit:   max age of object in fs.Object cache. mutually exlusive w/ -max-records
   -max-records: max number of objects in fs.Object cache. mutually exlusive w/ -age-limit
		`)
	}
	log.SetFlags(0)
}

//panics
func validateGcOptions() {
	ageopt := options.ageThreshold != infoAge(0)
	sizeopt := options.maxRecords != uint(0)
	if ageopt && sizeopt {
		panic("only one of age or size limits can be specified for the cache")
	} else if !(ageopt || sizeopt) {
		panic("one of age or size limits must be specified for the cache")
	}
}
func main() {

	//	defer panics.ExitHandler()

	flag.Parse()
	validateGcOptions()
	log.Println(about())

	opt := options
	var scout TrackScout = newTrackScout(opt.basepath, opt.pattern, uint16(opt.maxRecords), opt.ageThreshold)

	for {
		_, e := scout.Report()
		panics.OnError(e, "main", "scout.Report")

		time.Sleep(time.Millisecond * time.Duration(options.delaymsec))
	}

}

// ---- todo: extract lsf/fs

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
		maxAge            infoAge
		basepath, pattern string
	}
	objects *objcache
}

func newTrackScout(basepath, pattern string, maxSize uint16, maxAge infoAge) TrackScout {
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

	ageopt := t.options.maxAge != infoAge(0)
	sizeopt := t.options.maxSize != uint16(0)
	switch {
	case ageopt && sizeopt:
		panic("only one of age or size limits can be specified for the tracking scout object cache")
	case ageopt:
		t.objects = NewTimeWindowObjectCache(t.options.maxAge)
	case sizeopt:
		t.objects = NewFixedSizeObjectCache(t.options.maxSize)
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

// ----- TODO: extract lsf/fs -----------------------------

type objcache struct {
	options struct {
		maxSize uint16
		maxAge  infoAge
	}
	cache  map[string]fs.Object
	gc     GcFunc
	gcArgs []interface{}
}

var gcAlgorithm = struct{ byAge, bySize GcFunc }{
	byAge:  AgeBasedGcFunc,
	bySize: SizeBasedGcFunc,
}

func NewFixedSizeObjectCache(maxSize uint16) *objcache {
	oc := newObjectCache()
	oc.options.maxSize = maxSize
	oc.gc = SizeBasedGcFunc
	oc.gcArgs = []interface{}{oc.options.maxSize}
	return oc
}
func NewTimeWindowObjectCache(maxAge infoAge) *objcache {
	oc := newObjectCache()
	oc.options.maxAge = maxAge
	oc.gc = AgeBasedGcFunc
	oc.gcArgs = []interface{}{oc.options.maxAge}
	return oc
}
func newObjectCache() *objcache {
	oc := new(objcache)
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

	return IsDeleted0(obj.Flags()), nil
}

func IsDeleted0(flag uint8) bool {
	return flag == uint8(1)
}

func (oc *objcache) Gc() {
	n := oc.gc(oc.cache, oc.gcArgs...)
	if n > 0 {
		log.Printf("GC: %d items removed - object-cnt: %d", n, len(oc.cache))
	}
}

// REVU: TODO: sort these by age descending first
func AgeBasedGcFunc(cache map[string]fs.Object, args ...interface{}) int {
	panics.OnFalse(len(args) == 1, "BUG", "AgeBasedGcFunc", "args:", args)
	limit, ok := args[0].(infoAge)
	panics.OnFalse(ok, "BUG", "AgeBasedGcFunc", "limit:", args[0])
	n := 0
	for id, obj := range cache {
		if !IsDeleted0(obj.Flags()) {
			continue
		}
		if obj.Age() > time.Duration(limit) {
			delete(cache, id)
			n++
		}
	}
	return n
}

func SizeBasedGcFunc(cache map[string]fs.Object, args ...interface{}) int {
	panics.OnFalse(len(args) == 1, "BUG", "SizeBasedGcFunc", "args:", args)
	limit, ok := args[0].(uint16)
	panics.OnFalse(ok, "BUG", "SizeBasedGcFunc", "limit:", args[0])
	n := 0
	if len(cache) <= int(limit) {
		return 0
	}
	for id, obj := range cache {
		if !IsDeleted0(obj.Flags()) {
			continue
		}
		delete(cache, id)
		n++
	}
	return n
}

type GcFunc func(cache map[string]fs.Object, args ...interface{}) int
