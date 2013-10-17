// +build !windows

package main
import (
  "encoding/json"
  "os"
  "log"
)

func WriteRegistry(state map[string]*FileState, path string) {
  // Open tmp file, write, flush, rename
  file, err := os.Create(".lumberjack.new")
  if err != nil {
    log.Printf("Failed to open .lumberjack.new for writing: %s\n", err)
    return
  }
  defer file.Close()

  encoder := json.NewEncoder(file)
  encoder.Encode(state)

  os.Rename(".lumberjack.new", path)
}
