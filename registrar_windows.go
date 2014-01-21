package main

import (
  "encoding/json"
  "os"
  "log"
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
  if _, err := os.Stat(old); err == nil {
     os.Remove(old)
  }  
  os.Rename(path, old)
  os.Rename(tmp, path)
}

func NewFileState(info *os.FileInfo, source *string, offset int64) (fileState *FileState) {
  idxhi, idxlo, vol := FileIdentifiers(*info);
  return &FileState{Source: source, 
            Offset: offset, 
            Vol: vol,
            Idxhi: idxhi,
            Idxlo: idxlo}
}