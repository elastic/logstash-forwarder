package main

import (
	"flag"
	"log"
	. "lsf/capability"
	"lsf/fs"
	"lsf/panics"
	"os"
	"os/signal"
	"time"
)

var config struct {
	basepath, pattern string
	delay             time.Duration
}

var delayOpt uint

func init() {
	log.SetFlags(0)
	flag.StringVar(&config.basepath, "p", "", "path to log file dir")
	flag.StringVar(&config.pattern, "n", "*", "gob pattern")
	flag.UintVar(&delayOpt, "f", uint(100), "microsec delay between each log event")
}

func main() {
	flag.Parse()

	config.delay = time.Duration(delayOpt) * time.Microsecond
	if config.basepath == "" {
		log.Println("option -path is required.")
		flag.Usage()
		os.Exit(0)
	}

	log.Printf("ondemand-tracker: init: %s%s", config.basepath, config.pattern)
	ctl, requests, reports := tracker()

	user := make(chan os.Signal, 1)
	signal.Notify(user, os.Interrupt, os.Kill)

	go track(*ctl, requests, reports, config.basepath, config.pattern)

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

func tracker() (*control, chan struct{}, chan *TrackReport) {
	r := make(chan struct{}, 1)
	c := make(chan *TrackReport, 0)
	return procControl(), r, c
}

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

// ----------------------------------------------------------------------
// tracker task
// ----------------------------------------------------------------------
// TODO: extract config parameters for tracking capability
// TODO: fsobj_age_limit (for fsobject gc)
// TODO: fsobject_map_gc_threshold (for fsobject gc)
func track(ctl control, requests <-chan struct{}, out chan<- *TrackReport, basepath string, pattern string) {
	defer panics.AsyncRecover(ctl.stat, "done")

	log.Println("traking..")

	// maintains snapshot view of tracker after each request - initially empty
	//	var snapshot map[string]os.FileInfo = make(map[string]os.FileInfo)
	var snapshot map[string]fs.Object = make(map[string]fs.Object)

	// maintains historic list of all FS Objects we have seen, as
	// identified by fs.Object.Id() (and not the ephemeral filename)
	// This map is subject to garbage collection per configration params.
	var fsobjects map[string]fs.Object = make(map[string]fs.Object)

	// start report sequence counter - init is 0
	var reportSeq Counter = NewCounter()

	for {
		// block on requests
		select {
		case <-requests:
			// generate report
			report, snapshot1, e := reportFileEvents(basepath, pattern, snapshot, fsobjects, reportSeq)
			panics.OnError(e, "track", "reportFileEvents")

			// swap snapshot
			snapshot = snapshot1

			// update fsobjects - add any new fsobject
			for _, fileEvent := range report.Events {
				if fileEvent.Code == TrackEvent.NewFile {
					fsobj := fileEvent.File
					fsobjects[fsobj.Id()] = fsobj
				}
			}

			// publish report
			out <- report

			// gc fsobject map (if necessary)
			validDuration := time.Hour * 1  // REVU: 1HR for testing only TODO: config.fsobj_age_limit
			sizeThreshold := 100            // TODO: config.fsobject_map_gc_threshold
			if len(fsobjects) > sizeThreshold {
				log.Println("DEBUG TODO fsobjects size critical: %d", len(fsobjects))
				for _, fsobj := range fsobjects {
					if fsobj.Info().ModTime().Add(validDuration).Before(time.Now()) {
						delete(fsobjects, fsobj.Id())
						log.Printf("fsobj %s is garbage collected", fsobj.Id()) // TEMP DEBUG
					}
				}
			}
		}
	}
}
