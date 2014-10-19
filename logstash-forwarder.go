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
	configArg           string
	spoolSize           uint64
	harvesterBufferSize int
	cpuProfileFile      string
	idleTimeout         time.Duration
	useSyslog           bool
	tailOnRotate        bool
	debug               bool
	quiet               bool
}{
	spoolSize:           1024,
	harvesterBufferSize: 16 << 10,
	idleTimeout:         time.Second * 5,
}

func emitOptions() {
	emit("\t--- options -------\n")
	emit("\tconfig-arg:          %s\n", options.configArg)
	emit("\tidle-timeout:        %v\n", options.idleTimeout)
	emit("\tspool-size:          %d\n", options.spoolSize)
	emit("\tharvester-buff-size: %d\n", options.harvesterBufferSize)
	emit("\t--- flags ---------\n")
	emit("\ttail (on-rotation):  %t\n", options.tailOnRotate)
	emit("\tuse-syslog:          %t\n", options.useSyslog)
	emit("\tverbose:             %t\n", options.quiet)
	emit("\tdebug:               %t\n", options.debug)
	if runProfiler() {
		emit("\t--- profile run ---\n")
		emit("\tcpu-profile-file:    %s\n", options.cpuProfileFile)
	}

}

// exits with stat existStat.usageError if required options are not provided
func assertRequiredOptions() {
	if options.configArg == "" {
		exit(exitStat.usageError, "fatal: config file must be defined")
	}
}

const logflags = log.Ldate | log.Ltime | log.Lmicroseconds

var infolog *log.Logger

func init() {
	flag.StringVar(&options.configArg, "config", options.configArg, "path to logstash-forwarder configuration file or directory")

	flag.StringVar(&options.cpuProfileFile, "cpuprofile", options.cpuProfileFile, "path to cpu profile output - note: exits on profile end.")

	flag.Uint64Var(&options.spoolSize, "spool-size", options.spoolSize, "event count spool threshold - forces network flush")
	flag.Uint64Var(&options.spoolSize, "sv", options.spoolSize, "event count spool threshold - forces network flush")

	flag.IntVar(&options.harvesterBufferSize, "harvest-buffer-size", options.harvesterBufferSize, "harvester reader buffer size")
	flag.IntVar(&options.harvesterBufferSize, "hb", options.harvesterBufferSize, "harvester reader buffer size")

	flag.BoolVar(&options.useSyslog, "log-to-syslog", options.useSyslog, "log to syslog instead of stdout") // deprecate this
	flag.BoolVar(&options.useSyslog, "syslog", options.useSyslog, "log to syslog instead of stdout")

	flag.BoolVar(&options.tailOnRotate, "tail", options.tailOnRotate, "always tail on log rotation -note: may skip entries ")
	flag.BoolVar(&options.tailOnRotate, "t", options.tailOnRotate, "always tail on log rotation -note: may skip entries ")

	flag.BoolVar(&options.quiet, "verbose", options.quiet, "operate in quiet mode - only emit errors to log")
	flag.BoolVar(&options.quiet, "v", options.quiet, "operate in quiet mode - only emit errors to log")

	flag.BoolVar(&options.debug, "debug", options.debug, "emit debg info (verbose must also be set)")
}

func init() {
	infolog = log.New(os.Stdout, "", logflags)
	log.SetFlags(logflags)
}

func main() {
	defer func() {
		println("sanity")
		p := recover()
		if p == nil {
			return
		}
		fault("recovered panic: %v", p)
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

	config_files, err := DiscoverConfigs(options.configArg)
	if err != nil {
		fault("Could not use -config of '%s': %s", options.configArg, err)
	}

	var config Config

	for _, filename := range config_files {
		additional_config, err := LoadConfig(filename)
		if err == nil {
			err = MergeConfig(&config, additional_config)
		}
		if err != nil {
			fault("Could not load config file %s: %s", filename, err)
		}
	}
	FinalizeConfig(&config)

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
	// determine where in each file to restart a harvester.

//	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	if options.useSyslog {
		configureSyslog()
	}

	restart := &ProspectorResume{}
	restart.persist = make(chan *FileState)

	// Load the previous log file locations now, for use in prospector
	restart.files = make(map[string]*FileState)
	if existing, e := os.Open(".logstash-forwarder"); e == nil {
		defer existing.Close()
		wd := ""
		if wd, e = os.Getwd(); e != nil {
			emit("WARNING: os.Getwd retuned unexpected error %s -- ignoring\n", e.Error())
		}
		emit("Loading registrar data from %s/.logstash-forwarder\n", wd)

		decoder := json.NewDecoder(existing)
		decoder.Decode(&restart.files)
	}

	pendingProspectorCnt := 0

	// Prospect the globs/paths given on the command line and launch harvesters
	for _, fileconfig := range config.Files {
		prospector := &Prospector{FileConfig: fileconfig}
		go prospector.Prospect(restart, event_chan)
		pendingProspectorCnt++
	}

	// Now determine which states we need to persist by pulling the events from the prospectors
	// When we hit a nil source a prospector had finished so we decrease the expected events
	emit("Waiting for %d prospectors to initialise\n", pendingProspectorCnt)
	persist := make(map[string]*FileState)

	for event := range restart.persist {
		if event.Source == nil {
			pendingProspectorCnt--
			if pendingProspectorCnt == 0 {
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
}

// REVU: yes, this is a temp hack.
func emit(msgfmt string, args ...interface{}) {
	if options.quiet {
		return
	}
	infolog.Printf(msgfmt, args...)
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
