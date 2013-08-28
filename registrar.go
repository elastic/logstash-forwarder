package main

import (
  "log"
  "os"
  "syscall"
  "encoding/json"
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
      // have to dereference the FileInfo here because os.FileInfo is an
      // interface, not a struct, so Go doesn't have smarts to call the Sys()
      // method on a pointer to os.FileInfo. :(
      fstat := (*(event.fileinfo)).Sys().(*syscall.Stat_t)
      state[*event.Source] = &FileState{
        Source: event.Source,
        // take the offset + length of the line + newline char and
        // save it as the new starting offset.
        Offset: event.Offset + int64(len(*event.Text)) + 1,
        Inode: fstat.Ino,
        Device: uint64(fstat.Dev),
      }
    }

    if len(state) > 0 {
      write(state)
      os.Rename(".lumberjack.new", ".lumberjack")
    }
  }
}

func write(state map[string]*FileState) {
  log.Printf("Saving registrar state.\n")
  // Open tmp file, write, flush, rename
  file, err := os.Create(".lumberjack.new")
  if err != nil {
    log.Printf("Failed to open .lumberjack.new for writing: %s\n", err)
    return
  }

  encoder := json.NewEncoder(file)
  encoder.Encode(state)
  file.Close()
}
