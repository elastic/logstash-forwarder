package main

import (
  "fmt"
  "lumberjack"
  "os"
  "time"
  proto "code.google.com/p/goprotobuf/proto"
)

func emit(events []*lumberjack.FileEvent) {
  envelope := &lumberjack.EventEnvelope{Events: events}
  data, _ := proto.Marshal(envelope)
  fmt.Printf("Flushing !!! %d events: %d encoded\n", len(events), len(data))
} /* emit */

func main() {
  // TODO(sissel): support flags for setting... stuff
  event_chan := make(chan *lumberjack.FileEvent, 32)
  publisher_chan := make(chan *lumberjack.EventEnvelope, 1)

  // The basic model of execution:
  // - prospector: finds files in paths/globs to harvest, starts harvesters
  // - harvester: reads a file, sends events to the spooler
  // - spooler: buffers events until ready to flush to the publisher
  // - publisher: writes to the network, notifies registrar
  // - registrar: records positions of files read
  // Finally, prospector uses the registrar information, on restart, to
  // determine where in each file to resume a harvester.

  // TODO(sissel): need a prospector scan for files and launch harvesters

  // Prospect the globs/paths given on the command line and launch harvesters
  go lumberjack.Prospect(os.Args[1:], event_chan)

  var window_size uint64 = 1024 // Make this a flag
  var idle_timeout time.Duration = 1 * time.Second // Make this a flag

  // Harvesters dump events into the spooler.
  go lumberjack.Spooler(event_chan, publisher_chan, window_size, idle_timeout)

  // Spooler flushes periodically to the publisher
  for x := range publisher_chan {
    // got a bunch of events, ship them out.
    fmt.Printf("Spooler gave me %d events\n", len(x.Events))
    for _, event := range x.Events {
      fmt.Println(event)
    }
  }

  // TODO(sissel): publisher should send state to the registrar
  // TODO(sissel): registrar records last acknowledged positions in all files.
} /* main */
