// +build !windows

package main

import (
  "encoding/json"
  "log"
  "os"
)

func WriteRegistry(state map[string]*FileState, path string) {
  // Open tmp file, write, flush, rename
  file, err := os.Create(".logstash-forwarder.new")
  if err != nil {
    log.Printf("Failed to open .logstash-forwarder.new for writing: %s\n", err)
    return
  }
  defer file.Close()

  encoder := json.NewEncoder(file)
  encoder.Encode(state)

  os.Rename(".logstash-forwarder.new", path)
}
