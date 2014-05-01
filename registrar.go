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

      ino, dev := file_ids(event.fileinfo)
      state[*event.Source] = &FileState{
        Source: event.Source,
        // take the offset + length of the line + newline char and
        // save it as the new starting offset.
        Offset: event.Offset + int64(len(*event.Text)) + 1,
        Inode:  ino,
        Device: dev,
      }
      //log.Printf("State %s: %d\n", *event.Source, event.Offset)
    }

    if len(state) > 0 {
      WriteRegistry(state, ".logstash-forwarder")
    }
  }
}
