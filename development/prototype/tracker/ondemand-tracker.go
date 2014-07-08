/* THIS IS JUST A PROTOTYPE AND DEPRECATED - DON'T BOTHER TO FIX - was used to derive track func in lsfun */

package main

import (
	"flag"
	"github.com/elasticsearch/kriterium/panics"
	"log"
	"lsf/fs"
	. "lsf/lsfun"
	"os"
	"os/signal"
	"time"
)

var config struct {
	basepath, pattern    string
	delay                time.Duration
	fsObjectMaxAge       time.Duration
	fsObjectCacheMaxSize uint
}

type trackConfig struct {
	basepath, pattern    string
	fsObjectMaxAge       time.Duration
	fsObjectCacheMaxSize uint
}

var delayOpt, ageOpt uint

func init() {
	log.SetFlags(0)
	flag.StringVar(&config.basepath, "p", "", "path to log file dir")
	flag.StringVar(&config.pattern, "n", "*", "gob pattern")
	flag.UintVar(&delayOpt, "f", uint(100), "microsec delay between each log event")
	flag.UintVar(&ageOpt, "gc-age", uint(1), "fsobjects cache - max age in days")
	flag.UintVar(&config.fsObjectCacheMaxSize, "gc-size", uint(100), "fsobjects cache - gc threshold trigger")
}

func main() {
	flag.Parse()

	config.delay = time.Duration(delayOpt) * time.Microsecond
	config.fsObjectMaxAge = time.Duration(ageOpt) * time.Hour * 24
	config.delay = time.Duration(delayOpt) * time.Microsecond
	if config.basepath == "" {
		log.Println("option -path is required.")
		flag.Usage()
		os.Exit(0)
	}
	trackConfig := &trackConfig{config.basepath, config.pattern, config.fsObjectMaxAge, config.fsObjectCacheMaxSize}

	log.Printf("ondemand-tracker: init: %s%s", config.basepath, config.pattern)
	ctl, requests, reports := tracker()

	user := make(chan os.Signal, 1)
	signal.Notify(user, os.Interrupt, os.Kill)

	go track(*ctl, requests, reports, trackConfig)

	// driver
	flag := true
	requests <- struct{}{}
	for flag {
		select {
		case report := <-reports:
			if len(report.Events) > 0 {
				log.Println("\n" + report.String())
				for _, event := range report.Events {
					log.Println(event.String())
				}
				log.Println("\n" + report.String())
			}
			time.Sleep(config.delay)
			requests <- struct{}{}
		case stat := <-ctl.stat:
			log.Printf("stat: %s", stat)
			flag = false
		case <-user:
			os.Exit(0)
		default:
		}
	}
	// driver

	log.Println("bye")
	os.Exit(0)
}

// REVU: so what is 'c'? (n days later) TODO: be more explicit in var names here.
func tracker() (*control, chan struct{}, chan *TrackReport) {
	r := make(chan struct{}, 1)
	c := make(chan *TrackReport, 0)
	return procControl(), r, c
}

// REVU: TODO: use Control type from system/process
func procControl() *control {
	return &control{
		sig:  make(chan interface{}, 1),
		stat: make(chan interface{}, 1),
	}
}

type control struct {
	sig  chan interface{}
	stat chan interface{}
}

// TODO: lsf lib
// filters the os FS ignore list
func Ls(fspath string) ([]string, error) {
	file, e := os.Open(fspath)
	if e != nil {
		return nil, e
	}
	filenames, e := file.Readdirnames(0)
	if e != nil {
		return nil, e
	}
	e = file.Close()
	if e != nil {
		return nil, e
	}
	arr := make([]string, len(filenames))
	n := 0
	for _, filename := range filenames {
		if fs.Ignore(filename) {
			continue
		}
		arr[n] = filename
		n++
	}
	return arr[:n], nil
}

type Counter interface {
	Next() uint64
}

type counter struct {
	n uint64
}

func (c *counter) Next() (v uint64) { v = c.n; c.n++; return }
func NewCounter() Counter           { return &counter{} }

// TODO: refactor to fs.ReportFileEvents
// snapshot0 is current state record
// return report, new-snapshot, nil on success
// return nil, nil, error on error
// input args fspath, pattern, snapshot0, fsobjects are not modified.
func reportFileEvents(fspath, pattern string, snapshot0, fsobjects map[string]fs.Object, reportSeq Counter) (report *TrackReport, snapshot map[string]fs.Object, err error) {
	defer panics.Recover(&err)

	filenames, e := fs.FindMatchingPaths(fspath, pattern)
	panics.OnError(e, "trackingReport")

	// clone snapshot0 as working set
	workingset := make(map[string]fs.Object, len(snapshot0))
	for k, v := range snapshot0 {
		workingset[k] = v
	}
	snapshot = make(map[string]fs.Object)

	var events = make([]FileEvent, len(filenames)+len(snapshot0))
	var eventTime = time.Now()
	eventNum := 0

nextfile:
	for _, filename := range filenames {
		info, e := os.Stat(filename)
		switch {
		case e == nil:
		case os.IsNotExist(e): // were we tracking it?
			if fsobj, found := snapshot0[filename]; found {
				events[eventNum] = FileEvent{eventTime, TrackEvent.DeletedFile, fsobj}
				eventNum++
			}
			continue nextfile
		default: // permissions? // REVU: log it or what?
			log.Printf("DEBUG: ERROR: ignoring %s for %s", e.Error(), filename)
			continue nextfile
		}
		fsobj := fs.AsObject(info)

		// add to new snapshot
		snapshot[filename] = fsobj

		ssobj, found := workingset[filename]
		switch {
		case !found:
			events[eventNum] = FileEvent{eventTime, TrackEvent.NewFile, fsobj}
			eventNum++
		default:
			switch fs.SameObject(fsobj, ssobj) {
			case true:
				switch fs.Modified0(fsobj, ssobj) {
				case true:
					events[eventNum] = FileEvent{eventTime, TrackEvent.ModifiedFile, fsobj}
					eventNum++
				default:
				}
			default:
				var eventCode FileEventCode
				_, found := fsobjects[fsobj.Id()]
				switch found {
				case true:
					//					log.Printf("Object %s - RENAMED: %s to %s", fsobj.Id(), knownFsobj.Info().Name(), filename)
					eventCode = TrackEvent.RenamedFile
				default:
					eventCode = TrackEvent.NewFile
				}
				events[eventNum] = FileEvent{eventTime, eventCode, fsobj}
				eventNum++
			}
			delete(workingset, filename)
		}
	}

	// whatever is left in workingset was deleted
	for _, fsobj := range workingset {
		events[eventNum] = FileEvent{eventTime, TrackEvent.DeletedFile, fsobj}
		eventNum++
	}

	report = &TrackReport{reportSeq.Next(), fspath, events[:eventNum]}
	return

}

type TrackingScout interface {
	Report() (*TrackReport, error)
	Objects() []fs.Object
}
type scout struct {
	basepath string
	pattern  string
	sequence Counter
	objects  map[string]fs.Object
}

func NewTrackingScout(config *trackConfig) TrackingScout {
	s := new(scout)
	s.basepath = config.basepath
	s.pattern = config.pattern
	s.objects = make(map[string]fs.Object)
	s.sequence = NewCounter()
	return s
}
func (s *scout) Objects() []fs.Object {
	objects := make([]fs.Object, len(s.objects))
	n := 0
	for _, v := range s.objects {
		objects[n] = v
		n++
	}
	return objects
}
func (s *scout) Report() (report *TrackReport, err error) {
	// 2 tasks
	// 1: filter by glob and determine event (if any) for each file
	// 2: update fsobjects - we need it to assert file renames.
	// note: workingset/snapshot appears to be the active subset of snapshop
	//       if so, then size limit could bound the for..range over that cache
	//       and we won't have so many and maps to jugle!
	//
	// we need only this:
	// emit an fsobject on each request to tracking
	//
	// for report, we need:
	// emit a report listing events of type NEW, MODIFY, RENAME, DELETE

	panic("not implemented")
}

// ----------------------------------------------------------------------
// tracker task
// ----------------------------------------------------------------------

// TODO: extract config parameters for tracking lsfun
// TODO: fsobj_age_limit (for fsobject gc)
// TODO: fsobject_map_gc_threshold (for fsobject gc)
func track(ctl control, requests <-chan struct{}, out chan<- *TrackReport, config *trackConfig) {
	defer panics.AsyncRecover(ctl.stat, "done")

	log.Println("traking..")

	var scout TrackingScout = NewTrackingScout(config)
	var tracker TrackingAnalysis = NewTrackingAnalysis(config)

	//	// maintains snapshot view of tracker after each request - initially empty
	//	//	var snapshot map[string]os.FileInfo = make(map[string]os.FileInfo)
	//	var snapshot map[string]fs.Object = make(map[string]fs.Object)
	//
	//	// maintains historic list of all FS Objects we have seen, as
	//	// identified by fs.Object.Id() (and not the ephemeral filename)
	//	// This map is subject to garbage collection per configration params.
	//	var fsobjects map[string]fs.Object = make(map[string]fs.Object)
	//
	//	// start report sequence counter - init is 0
	//	var reportSeq Counter = NewCounter()

	//	var trackedObj fs.Object

next:
	for {
		// block on requests
		select {
		case <-requests:
			// generate report
			// TODO: insure returned snapshot is effectively ignorable (same as snapshot) if events len is 0
			report, e := scout.Report()
			//			report, snapshot1, e := reportFileEvents(config.basepath, config.pattern, snapshot, fsobjects, reportSeq)
			panics.OnError(e, "track", "reportFileEvents")
			if len(report.Events) == 0 {
				// publish 0 len report
				out <- report
				continue next
			}
			out <- report

			selection, e := tracker.Update(snapshot1, report)
			panics.OnError(e, "track", "tracker.Update")

			log.Printf("PICK: %s\n", selection)

			//			// TODO: refactor to trackAnalysis
			//			// REVU: TODO: n TRK events need to be reduced to 1 per stream def
			//			// swap snapshot
			//			snapshot = snapshot1
			//
			//			// update fsobjects - add any new fsobject
			//			// select fsobj to track
			//			var youngest time.Duration = time.Hour * 24 * 365 * 100
			//			var candidateEvent FileEvent
			//			var candidateObj fs.Object
			//			for _, fileEvent := range report.Events {
			//				fsobj := fileEvent.File
			//				if fileEvent.Code == TrackEvent.NewFile {
			//					fsobjects[fsobj.Id()] = fsobj
			//				}
			//				if fsobj.Age() < youngest {
			//					youngest = fsobj.Age()
			//					candidateObj = fsobj
			//					candidateEvent = fileEvent
			//				}
			//			}
			//
			//			if trackedObj == nil {
			//				trackedObj = candidateObj
			//			} else if !fs.SameObject(trackedObj, candidateObj) {
			//				// If candidateEvent is TRK AND trackedObj.Age() > candidateObj.Age()
			//				if candidateEvent.Code == TrackEvent.NewFile {
			//					if trackedObj.Age() > candidateObj.Age() {
			//						trackedObj = candidateObj
			//					}
			//				} else {
			//					// is this possible? exists in map but younger than what is tracked?
			//					// TODO: needs REVU for how to handle it.
			//					panics.OnTrue(true, candidateEvent.Code, "candidate:", candidateObj, "tracked:", trackedObj)
			//				}
			//			}
			//
			//			out <- report
			//
			//			// gc fsobject map (if necessary)
			//			if len(fsobjects) > int(config.fsObjectCacheMaxSize) {
			//				log.Println("DEBUG TODO fsobjects size critical: %d", len(fsobjects))
			//				for _, fsobj := range fsobjects {
			//					if fsobj.Age() > config.fsObjectMaxAge {
			//						delete(fsobjects, fsobj.Id())
			//						log.Printf("fsobj %s is garbage collected", fsobj.Id()) // TEMP DEBUG
			//					}
			//				}
			//			}
		}
	}
}

// tracker performs analysis on TrackReports and makes tracking decisions.
type TrackingAnalysis interface {
	Update(snapshot map[string]fs.Object, trackReport *TrackReport) (fs.Object, error)
	Select() fs.Object
}

type trackingInfo struct {
	// currently selected fsobject
	trackedObj fs.Object

	// maintains historic list of all FS Objects we have seen, as
	// identified by fs.Object.Id() (and not the ephemeral filename)
	// This map is subject to garbage collection per configration params.
	fsobjects map[string]fs.Object

	// maintains snapshot view of tracker after each request - initially empty
	//	var snapshot map[string]os.FileInfo = make(map[string]os.FileInfo)
	snapshot map[string]fs.Object
}

func (t *trackingInfo) Select() fs.Object {
	panic("not implemented")
}

func (t *trackingInfo) Update(snapshot1 map[string]fs.Object, report *TrackReport) (selection fs.Object, err error) {
	defer panics.Recover(&err)

	if len(report.Events) == 0 {
		return t.trackedObj, nil
	}
	// TODO: refactor to trackAnalysis
	// REVU: TODO: n TRK events need to be reduced to 1 per stream def
	// swap snapshot
	t.snapshot = snapshot1
	//	snapshot = snapshot1

	// update fsobjects - add any new fsobject
	// select fsobj to track
	var youngest time.Duration = time.Hour * 24 * 365 * 100
	var candidateEvent FileEvent
	var candidateObj fs.Object
	for _, fileEvent := range (*report).Events {
		fsobj := fileEvent.File
		if fileEvent.Code == TrackEvent.NewFile {
			t.fsobjects[fsobj.Id()] = fsobj
		}
		if fsobj.Age() < youngest {
			youngest = fsobj.Age()
			candidateObj = fsobj
			candidateEvent = fileEvent
		}
	}

	if t.trackedObj == nil {
		t.trackedObj = candidateObj
	} else if !fs.SameObject(t.trackedObj, candidateObj) {
		// If candidateEvent is TRK AND trackedObj.Age() > candidateObj.Age()
		if candidateEvent.Code == TrackEvent.NewFile {
			if t.trackedObj.Age() > candidateObj.Age() {
				t.trackedObj = candidateObj
			}
		} else {
			// is this possible? exists in map but younger than what is tracked?
			// TODO: needs REVU for how to handle it.
			panics.OnTrue(true, candidateEvent.Code, "candidate:", candidateObj, "tracked:", t.trackedObj)
		}
	}

	// gc fsobject map (if necessary)
	if len(t.fsobjects) > int(config.fsObjectCacheMaxSize) {
		log.Println("DEBUG TODO fsobjects size critical: %d", len(t.fsobjects))
		for _, fsobj := range t.fsobjects {
			if fsobj.Age() > config.fsObjectMaxAge {
				delete(t.fsobjects, fsobj.Id())
				log.Printf("fsobj %s is garbage collected", fsobj.Id()) // TEMP DEBUG
			}
		}
	}
	return t.trackedObj, nil
}

func NewTrackingAnalysis(config *trackConfig) TrackingAnalysis {
	t := new(trackingInfo)
	t.fsobjects = make(map[string]fs.Object)
	t.snapshot = make(map[string]fs.Object)
	return t
}
