package main

import (
	"flag"
	"fmt"
	"log"
	"lsf/fs"
	"lsf/lsfun"
	"lsf/panics"
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
	var scout lsfun.TrackScout = lsfun.NewTrackScout(opt.basepath, opt.pattern, uint16(opt.maxSize), opt.maxAge)

	for {
		report, e := scout.Report()
		panics.OnError(e, "main", "scout.Report")

		// REVU: TODO: this need to be emitted via a RolloverLogWriter(event)
		for _, event := range report.Events {
			if event.Code != lsfun.TrackEvent.KnownFile { // printing NOP events gets noisy
				log.Println(event)
			}
		}

		time.Sleep(time.Millisecond * time.Duration(options.delaymsec))
	}

}
