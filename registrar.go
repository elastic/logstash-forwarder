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

      ino, dev := file_ids(event.fileinfo)
      state[*event.Source] = &FileState{
        Source: event.Source,
        // take the offset + length of the line + newline char and
        // save it as the new starting offset.
        // This issues a problem, if the EOL is a CRLF! Then on start it read the LF again and generates a event with an empty line
        Offset: event.Offset + int64(len(*event.Text)) + 1,
        Inode:  ino,
        Device: dev,
      }
      //log.Printf("State %s: %d\n", *event.Source, event.Offset)
    }

    WriteRegistry(state, ".logstash-forwarder")
  }
}
