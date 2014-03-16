package main

import (
  "log"
)

func Registrar(new_state map[string]*FileState, input chan []*FileEvent) {
  state := new_state
  for events := range input {
    log.Printf("Registrar received %d events\n", len(events))
    // Take the last event found for each file source
    for _, event := range events {
      // skip stdin
      if *event.Source == "-" {
        continue
      }
      // have to dereference the FileInfo here because os.FileInfo is an
      // interface, not a struct, so Go doesn't have smarts to call the Sys()
      // method on a pointer to os.FileInfo. :(
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
