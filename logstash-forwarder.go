package main

import (
  "encoding/json"
  "flag"
  "log"
  "os"
  "runtime/pprof"
  "time"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var spool_size = flag.Uint64("spool-size", 1024, "Maximum number of events to spool before a flush is forced.")
var idle_timeout = flag.Duration("idle-flush-time", 5*time.Second, "Maximum time to wait for a full spool before flushing anyway")
var config_file = flag.String("config", "", "The config file to load")
var use_syslog = flag.Bool("log-to-syslog", false, "Log to syslog instead of stdout")
var from_beginning = flag.Bool("from-beginning", false, "Read new files from the beginning, instead of the end")

func main() {
  flag.Parse()

  if *cpuprofile != "" {
    f, err := os.Create(*cpuprofile)
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

  config, err := LoadConfig(*config_file)
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
  if *use_syslog {
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
    log.Printf("Loading registrar data from %s/.logstash-forwarder\n", wd)

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
  log.Printf("Waiting for %d prospectors to initialise\n", prospector_pending)
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
    log.Printf("Registrar will re-save state for %s\n", *event.Source)
  }

  log.Printf("All prospectors initialised with %d states to persist\n", len(persist))

  // Harvesters dump events into the spooler.
  go Spool(event_chan, publisher_chan, *spool_size, *idle_timeout)

  go Publishv1(publisher_chan, registrar_chan, &config.Network)

  // registrar records last acknowledged positions in all files.
  Registrar(persist, registrar_chan)
} /* main */
