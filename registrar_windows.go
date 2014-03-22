package main

import (
  "encoding/json"
  "log"
  "os"
)

func WriteRegistry(state map[string]*FileState, path string) {
  tmp := path + ".new"
  file, err := os.Create(tmp)
  if err != nil {
    log.Printf("Failed to open .logstash-forwarder.new for writing: %s\n", err)
    return
  }

  encoder := json.NewEncoder(file)
  encoder.Encode(state)
  file.Close()

  old := path + ".old"

  if _, err = os.Stat(old); err != nil && os.IsNotExist(err) {
  } else {
    err = os.Remove(old)
    if err != nil {
      log.Printf("Registrar save problem: Failed to delete backup file: %s\n", err)
    }
  }

  if _, err = os.Stat(path); err != nil && os.IsNotExist(err) {
  } else {
    err = os.Rename(path, old)
    if err != nil {
      log.Printf("Registrar save problem: Failed to perform backup: %s\n", err)
    }
  }

  err = os.Rename(tmp, path)
  if err != nil {
    log.Printf("Registrar save problem: Failed to move the new file into place: %s\n", err)
  }
}
