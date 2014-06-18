package main

import (
	"flag"
	"log"
	. "lsf/capability"
	"lsf/fs"
	"lsf/panics"
	"os"
	"os/signal"
	"path"
	"time"
)

var config struct {
	path  string
	delay time.Duration
}

var delayOpt uint

func init() {
	log.SetFlags(0)
	flag.StringVar(&config.path, "path", "", "path to log file dir")
	flag.UintVar(&delayOpt, "f", uint(100), "microsec delay between each log event")
}

//var publish chan interface{}

func main() {
	flag.Parse()
	config.delay = time.Duration(delayOpt) * time.Microsecond
	if config.path == "" {
		log.Println("option -path is required.")
		flag.Usage()
		os.Exit(0)
	}

	log.Printf("trak %q", config.path)
	ctl, requests, reports := tracker()

	user := make(chan os.Signal, 1)
	signal.Notify(user, os.Interrupt, os.Kill)

	go track(*ctl, requests, reports, config.path, "*")

	// driver
	flag := true
	requests <- struct{}{}
	for flag {
		//		time.Sleep(time.Microsecond)
		select {
		case report := <-reports:
			log.Println("\n" + report.String())
			for _, event := range report.Events {
				log.Println(event.String())
			}
			// wait a bit before requesting next update
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

//
// snapshot0 is current state record
// return report, new-snapshot, nil on success
// return nil, nil, error on error
func reportFileEvents(fspath, pattern string, snapshot0 map[string]fs.Object, reportSeq Counter) (report *TrackReport, snapshot map[string]fs.Object, err error) {
	defer panics.Recover(&err)

	filenames, e := Ls(fspath)
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

	//nextfile:
	for _, basename := range filenames {

		// assert file (still) exists - if not: delete event
		filename := path.Join(fspath, basename)
		info, e := os.Stat(filename)
		if e != nil {
			// deleted under our nose - were we tracking it?
			if fsobj, found := snapshot0[basename]; found {
				events[eventNum] = FileEvent{eventTime, TrackEvent.DeletedFile, fsobj}
				eventNum++
				continue
			}
		}
		fsobj := fs.AsObject(info)

		// add to new snapshot
		snapshot[basename] = fsobj

		ssobj, found := workingset[basename]
		switch {
		case !found:
			events[eventNum] = FileEvent{eventTime, TrackEvent.NewFile, fsobj}
			eventNum++
		default:
			if fs.SameObject(fsobj, ssobj) {
				if fs.Modified0(fsobj, ssobj) { // change
					events[eventNum] = FileEvent{eventTime, TrackEvent.ModifiedFile, fsobj}
					eventNum++
					delete(workingset, basename)
				} else { // static
					delete(workingset, basename)
				}
			} else {
				// found, not same object but shared name at some time .. ? swap
				events[eventNum] = FileEvent{eventTime, TrackEvent.RenamedFile, fsobj}
				eventNum++
				delete(workingset, basename)
			}
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
func track(ctl control, requests <-chan struct{}, out chan<- *TrackReport, basepath string, pattern string) {
	defer panics.AsyncRecover(ctl.stat, "done")

	log.Println("traking..")

	// maintains snapshot view of tracker after each request - initially empty
	//	var snapshot map[string]os.FileInfo = make(map[string]os.FileInfo)
	var snapshot map[string]fs.Object = make(map[string]fs.Object)

	// TODO: this should be passed in ..
	var reportSeq Counter = NewCounter()

	for {
		select {
		case <-requests:
			report, snapshot1, e := reportFileEvents(basepath, pattern, snapshot, reportSeq)
			panics.OnError(e, "")
			// swap snapshot and send the report
			snapshot = snapshot1
			out <- report
		}
	}
}

func trackingAnalysis(snapshot, workingSet map[string]fs.Object) []FileEvent {

	return nil
}
