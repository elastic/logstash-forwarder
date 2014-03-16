package main

import (
  "os"
)

type FileState struct {
  Source *string `json:"source,omitempty"`
  Offset int64   `json:"offset,omitempty"`
  Vol    uint32  `json:"vol,omitempty"`
  IdxHi  uint32  `json:"idxhi,omitempty"`
  IdxLo  uint32  `json:"idxlo,omitempty"`
}

func file_ids(info *os.FileInfo, state *FileState) {
  fstat := (*info).(*fileStat)
  e := fstat.loadFileId()
  if e != nil {
    return
  }
  state.Vol = fstat.vol
  state.IdxHi = fstat.idxhi
  state.IdxLo = fstat.idxlow
}

func is_file_same(path string, info os.FileInfo, state *FileState) bool {
  istate := &FileState{}
  file_ids(info, istate)
  return (istate.Vol == state.Vol && istate.IdxHi == state.IdxHi && istate.IdxLo == state.IdxLo)
}
