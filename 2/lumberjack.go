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
  // TODO(sissel): need a HarvestForeman to manage the harvester
  // TODO(sissel): Need an encryptor
  // TODO(sissel): Need a compressor
  event_stream := make(chan *lumberjack.FileEvent, 32)
  h := lumberjack.Harvester{Path: "/var/log/messages"}
  go h.Harvest(event_stream)
  go h.Harvest(event_stream)

  //flusher = func(events []interface{}) {
    //fmt.Printf("Flushing %d events\n", len(events))
  //}
  
  var window_size uint64 = 1024
  timeout := 1 * time.Second
  emitter_stream := make(chan *lumberjack.EventEnvelope, 1)
  // harvester -> spooler
  go lumberjack.Spooler(event_stream, emitter_stream, window_size, timeout)

  // spooler -> emitter
  for x := range emitter_stream {
    // got a bunch of events, ship them out.
    fmt.Printf("Spooler gave me %d events\n", len(x.Events))
  }
  //lumberjack.Emitter(
} /* main */
