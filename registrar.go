package main

import (
  "log"
)

func Registrar(input chan []*FileEvent) {
  for events := range input {
    state := make(map[string]*FileState)
    log.Printf("Registrar received %d events\n", len(events))
    // Take the last event found for each file source
    for _, event := range events {
      // skip stdin
      if *event.Source == "-" {
        continue
      }

      state[*event.Source] = NewFileState(event.fileinfo, 
      									  event.Source, 
      									  event.Offset + int64(len(*event.Text)) + 1) 
//      log.Printf("State %s: %d\n", *event.Source, event.Offset)
    }

    if len(state) > 0 {
      WriteRegistry(state, ".logstash-forwarder")
    }
  }
}

