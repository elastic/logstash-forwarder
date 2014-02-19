// +build !windows

package main

import (
  "encoding/json"
  "log"
  "os"
)

func WriteRegistry(state map[string]*FileState, path string) {
  // Open tmp file, write, flush, rename
  file, err := os.Create("/var/lib/logstash-forwarder.state.new")
  if err != nil {
    log.Printf("Failed to open /var/lib/logstash-forwarder.state.new for writing: %s\n", err)
    return
  }
  defer file.Close()

  encoder := json.NewEncoder(file)
  encoder.Encode(state)

  os.Rename("/var/lib/logstash-forwarder.state.new", path)
}
