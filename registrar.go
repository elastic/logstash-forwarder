package main

import (
  "log"
)

func Registrar(state map[string]*FileState, input chan []*FileEvent) {
  for events := range input {
    log.Printf("Registrar received %d events\n", len(events))
    // Take the last event found for each file source
    for _, event := range events {
      // skip stdin
      if *event.Source == "-" {
        continue
      }
      state[*event.Source] = &FileState{
        Source: event.Source,
        Offset: event.Offset,
      }
      file_ids(event.fileinfo, state[*event.Source])
      //log.Printf("State %s: %d\n", *event.Source, event.Offset)
    }

    if len(state) > 0 {
      WriteRegistry(state, ".logstash-forwarder")
    }
  }
}
