package lumberjack

import (
  "fmt"
)

func Publish(input chan *EventEnvelope) {
  // Spooler flushes periodically to the publisher
  for envelope := range input {
    // got a bunch of events, ship them out.
    fmt.Printf("Spooler gave me %d events\n", len(envelope.Events))

    // TODO(sissel): serialize to string (proto.Marshal or whatever)
    // TODO(sissel): compress (zlib?)
    // TODO(sissel): encrypt (aes?)
    // TODO(sissel): send out over zeromq REQ/REP
    // TODO(sissel): retry on failure or timeout
    // TODO(sissel): notify registrar of success
  }
} // Publish
