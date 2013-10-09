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
    log.Printf("Registrar received %d events for %s\n", len(events), *events[0].Source)
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
        Device: fstat.Dev,
      }
    }

    if len(state) > 0 {
      write(state)
    }
  }
}

func write(state map[string]*FileState) {
  // Open tmp file, write, flush, rename
  log.Printf("Saving registrar state.\n")

  //read the current state file and overwrite the current state vaules
  historical_state := make(map[string]*FileState)
  history, err := os.Open(".lumberjack")
  if err != nil {
    log.Printf("Registar was unable to read privous states. Error: %s\n", err)
    return
  } else {
    decoder := json.NewDecoder(history)
    decoder.Decode(&historical_state)
    history.Close()

  }

    //loop though the sate for the file, should be a map of lenght 1
    for path, new_state := range state {
      historical_state[path] = new_state
    }

    file, err := os.Create(".lumberjack.new")
    if err != nil {
      log.Printf("Failed to open .lumberjack.new for writing new state: %s\n", err)
      return
    }

    encoder := json.NewEncoder(file)
    encoder.Encode(historical_state)
    file.Close()

  os.Rename(".lumberjack.new", ".lumberjack")

}
