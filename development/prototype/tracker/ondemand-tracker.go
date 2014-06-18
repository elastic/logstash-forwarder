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
	log.Printf("%d", delayOpt)
	log.Printf("%d", config.delay.Nanoseconds())
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

func tracker() (*control, chan struct{}, chan *Trackreport) {
	r := make(chan struct{}, 1)
	c := make(chan *Trackreport, 0)
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

// ----------------------------------------------------------------------
// tracker task
// ----------------------------------------------------------------------
func track(ctl control, requests <-chan struct{}, out chan<- *Trackreport, basepath string, pattern string) {
	defer panics.AsyncRecover(ctl.stat, "done")

	log.Println("traking..")

	// maintains snapshot view of tracker after each request - initially empty
	//	var snapshot map[string]os.FileInfo = make(map[string]os.FileInfo)
	var snapshot map[string]fs.Object = make(map[string]fs.Object)

	for {
		select {
		case <-requests:

			file, e := os.Open(basepath)
			panics.OnError(e)
			filenames, e := file.Readdirnames(0)
			panics.OnError(e)
			e = file.Close()
			panics.OnError(e)

			workingset := make(map[string]fs.Object)

			var eventTime = time.Now()
			var eventType FileEventCode
			var events = make([]FileEvent, len(filenames)+len(snapshot))
			var eventNum = 0

			for _, basename := range filenames {
				if fs.Ignore(basename) {
					continue
				}

				filename := path.Join(basepath, basename)
				info, e := os.Stat(filename)
				if e != nil {
					// deleted under our nose
					// were we tracking it?
					if fsobj, found := snapshot[basename]; found {
						events[eventNum] = FileEvent{eventTime, TrackEvent.DeletedFile, fsobj}
						eventNum++
						delete(snapshot, basename)
						continue
					}
				}
				fsobj := fs.AsObject(info)
				workingset[basename] = fsobj

				obj0 := snapshot[basename]
				news := false
				// is it news?
				if obj0 != nil {
					// compare
					if obj0.Info().Size() != info.Size() {
						// changed
						eventType = TrackEvent.ModifiedFile
						news = true
					}
				} else {
					eventType = TrackEvent.NewFile
					news = true
				}
				if news {
					events[eventNum] = FileEvent{eventTime, eventType, fsobj}
					eventNum++
					snapshot[basename] = fsobj
				}
			}
			// were we tracking anything that is no longer in the dir?
			for f, _ := range snapshot {
				if _, found := workingset[f]; !found {
					events[eventNum] = FileEvent{eventTime, TrackEvent.DeletedFile, snapshot[f]}
					eventNum++
					delete(snapshot, f)
				}
			}
			report := Trackreport{basepath, events[:eventNum]}
			out <- &report
		}
	}
}

func trackingAnalysis(snapshot, workingSet map[string]fs.Object) []FileEvent {

	return nil
}
