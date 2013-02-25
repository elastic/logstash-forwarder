package main

import (
  "fmt"
  "lumberjack"
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
  emitter_chan := make(chan *lumberjack.EventEnvelope, 1)

  // The basic model of execution:
  // - prospector: finds files in paths/globs to harvest, starts harvesters
  // - harvester: reads a file, sends events to the spooler
  // - spooler: buffers events until ready to flush to the emitter
  // - emitter: writes to the network, notifies registrar
  // - registrar: records positions of files read
  // Finally, prospector uses the registrar information, on restart, to
  // determine where in each file to resume a harvester.

  // TODO(sissel): need a prospector scan for files and launch harvesters

  // Example dummy harvester
  h := lumberjack.Harvester{Path: "/var/log/messages"}
  go h.Harvest(event_chan)
  go h.Harvest(event_chan)

  var window_size uint64 = 1024 // Make this a flag
  var idle_timeout time.Duration = 1 * time.Second // Make this a flag

  // harvester -> spooler
  go lumberjack.Spooler(event_chan, emitter_chan, window_size, idle_timeout)

  // spooler -> publisher
  for x := range emitter_chan {
    // got a bunch of events, ship them out.
    fmt.Printf("Spooler gave me %d events\n", len(x.Events))
  }

  // emitter should send acknowledgements to the registrar
  // registrar records last acknowledged positions in all files.
} /* main */
