package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"runtime/pprof"
	"time"
)

var exitStat = struct {
	ok, usageError, faulted int
}{
	ok:         0,
	usageError: 1,
	faulted:    2,
}

var options = &struct {
	configFile     string
	spoolSize      uint64
	cpuProfileFile string
	idleTimeout    time.Duration
	useSyslog      bool
	tailOnRotate   bool
	debug          bool
	verbose        bool
}{
	spoolSize:   1024,
	idleTimeout: time.Second * 5,
}

func emitOptions() {
	emit("\t--- options -------\n")
	emit("\tconfig-file:        %s\n", options.configFile)
	emit("\tidle-timeout:       %v\n", options.idleTimeout)
	emit("\t--- flags ---------\n")
	emit("\ttail (on-rotation): %t\n", options.tailOnRotate)
	emit("\tuse-syslog:         %t\n", options.useSyslog)
	emit("\tverbose:            %t\n", options.verbose)
	emit("\tdebug:              %t\n", options.debug)
	if runProfiler() {
		emit("\t--- profile run ---\n")
		emit("\tcpu-profile-file: %s\n", options.cpuProfileFile)
	}

}

// exits with stat existStat.usageError if required options are not provided
func assertRequiredOptions() {
	if options.configFile == "" {
		exit(exitStat.usageError, "fatal: config file must be defined")
	}
}

func init() {
	flag.StringVar(&options.configFile, "config", options.configFile, "path to logstash-forwarder configuration file")
	flag.StringVar(&options.cpuProfileFile, "cpuprofile", options.cpuProfileFile, "path to cpu profile output - note: exits on profile end.")
	flag.Uint64Var(&options.spoolSize, "spool-size", options.spoolSize, "event count spool threshold - forces network flush")
	flag.BoolVar(&options.useSyslog, "log-to-syslog", options.useSyslog, "log to syslog instead of stdout")
	flag.BoolVar(&options.tailOnRotate, "tail", options.tailOnRotate, "always tail on log rotation -note: may skip entries ")
	flag.BoolVar(&options.tailOnRotate, "t", options.tailOnRotate, "always tail on log rotation -note: may skip entries ")
	flag.BoolVar(&options.verbose, "verbose", options.verbose, "operate in verbose mode - emits to log")
	flag.BoolVar(&options.verbose, "v", options.verbose, "operate in verbose mode - emits to log")
	flag.BoolVar(&options.debug, "debug", options.debug, "emit debg info (verbose must also be set)")
}

func main() {
	defer func() {
		println("sanity")
		p := recover()
		if p == nil {
			return
		}
		log.Fatalf("panic: %v\n", p)
	}()

	flag.Parse()
	assertRequiredOptions()
	emitOptions()

	if runProfiler() {
		f, err := os.Create(options.cpuProfileFile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		go func() {
			time.Sleep(60 * time.Second)
			pprof.StopCPUProfile()
			panic("done")
		}()
	}

	config, err := LoadConfig(options.configFile)
	if err != nil {
		return
	}

	event_chan := make(chan *FileEvent, 16)
	publisher_chan := make(chan []*FileEvent, 1)
	registrar_chan := make(chan []*FileEvent, 1)

	if len(config.Files) == 0 {
		log.Fatalf("No paths given. What files do you want me to watch?\n")
	}

	// The basic model of execution:
	// - prospector: finds files in paths/globs to harvest, starts harvesters
	// - harvester: reads a file, sends events to the spooler
	// - spooler: buffers events until ready to flush to the publisher
	// - publisher: writes to the network, notifies registrar
	// - registrar: records positions of files read
	// Finally, prospector uses the registrar information, on restart, to
	// determine where in each file to resume a harvester.

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	if options.useSyslog {
		configureSyslog()
	}

	resume := &ProspectorResume{}
	resume.persist = make(chan *FileState)

	// Load the previous log file locations now, for use in prospector
	resume.files = make(map[string]*FileState)
	history, err := os.Open(".logstash-forwarder")
	if err == nil {
		wd, err := os.Getwd()
		if err != nil {
			wd = ""
		}
		emit("Loading registrar data from %s/.logstash-forwarder\n", wd)

		decoder := json.NewDecoder(history)
		decoder.Decode(&resume.files)
		history.Close()
	}

	prospector_pending := 0

	// Prospect the globs/paths given on the command line and launch harvesters
	for _, fileconfig := range config.Files {
		prospector := &Prospector{FileConfig: fileconfig}
		go prospector.Prospect(resume, event_chan)
		prospector_pending++
	}

	// Now determine which states we need to persist by pulling the events from the prospectors
	// When we hit a nil source a prospector had finished so we decrease the expected events
	emit("Waiting for %d prospectors to initialise\n", prospector_pending)
	persist := make(map[string]*FileState)

	for event := range resume.persist {
		if event.Source == nil {
			prospector_pending--
			if prospector_pending == 0 {
				break
			}
			continue
		}
		persist[*event.Source] = event
		emit("Registrar will re-save state for %s\n", *event.Source)
	}

	emit("All prospectors initialised with %d states to persist\n", len(persist))

	// Harvesters dump events into the spooler.
	go Spool(event_chan, publisher_chan, options.spoolSize, options.idleTimeout)

	go Publishv1(publisher_chan, registrar_chan, &config.Network)

	// registrar records last acknowledged positions in all files.
	Registrar(persist, registrar_chan)
} /* main */

// REVU: yes, this is a temp hack.
func emit(msgfmt string, args ...interface{}) {
	if !options.verbose {
		return
	}
	log.Printf(msgfmt, args...)
}

func fault(msgfmt string, args ...interface{}) {
	exit(exitStat.faulted, msgfmt, args...)
}
func exit(stat int, msgfmt string, args ...interface{}) {
	log.Printf(msgfmt, args...)
	os.Exit(stat)
}

func runProfiler() bool {
	return options.cpuProfileFile != ""
}
