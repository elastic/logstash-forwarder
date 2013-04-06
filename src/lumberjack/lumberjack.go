package main

import (
  "log"
  lumberjack "liblumberjack"
  "os"
  "time"
  "flag"
  "strings"
  "runtime/pprof"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

var spool_size = flag.Uint64("spool-size", 1024, "Maximum number of events to spool before a flush is forced.")
var idle_timeout = flag.Duration("idle-flush-time", 5 * time.Second, "Maximum time to wait for a full spool before flushing anyway")
var server_timeout = flag.Duration("server-timeout", 30 * time.Second, "Maximum time to wait for a request to a server before giving up and trying another.")
var servers = flag.String("servers", "", "Server (or comma-separated list of servers) to send events to. Each server can be a 'host' or 'host:port'. If the port is not specified, port 5005 is assumed. One server is chosen of the list at random, and only on failure is another server used.")

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

  // Turn 'host' and 'host:port' into 'tcp://host:port'
  if *servers == "" {
    log.Printf("No servers specified, please provide the -servers setting\n")
    return
  }
  server_list := strings.Split(*servers, ",")
  for i, server := range server_list {
    if !strings.Contains(server, ":") {
      server_list[i] = "tcp://" + server + ":5005"
    } else {
      server_list[i] = "tcp://" + server
    }
  }

  log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

  // TODO(sissel): support flags for setting... stuff
  event_chan := make(chan *lumberjack.FileEvent, 16)
  publisher_chan := make(chan []*lumberjack.FileEvent, 1)

  // The basic model of execution:
  // - prospector: finds files in paths/globs to harvest, starts harvesters
  // - harvester: reads a file, sends events to the spooler
  // - spooler: buffers events until ready to flush to the publisher
  // - publisher: writes to the network, notifies registrar
  // - registrar: records positions of files read
  // Finally, prospector uses the registrar information, on restart, to
  // determine where in each file to resume a harvester.

  // Prospect the globs/paths given on the command line and launch harvesters
  go lumberjack.Prospect(flag.Args(), event_chan)

  // Harvesters dump events into the spooler.
  go lumberjack.Spool(event_chan, publisher_chan, *spool_size, *idle_timeout)

  lumberjack.Publish(publisher_chan, server_list, *server_timeout)

  // TODO(sissel): publisher should send state to the registrar
  // TODO(sissel): registrar records last acknowledged positions in all files.
} /* main */
