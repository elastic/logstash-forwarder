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
  // TODO(golang): Make the following Windows fileStat struct members accessible somehow: vol, idxhi, idxlo
  //               They are the struct members used for samefile - the equivilant to device and inode on Linux
  //               At the moment they are truly unreachable due to Go's package separation
  //               Rather than reinvent the wheel we just need to wait for an interface to them
  //               Sys() returns the WIN32_FILE_ATTRIBUTE_DATA unfortunately which is not what we need

  // Until the above TODO is completely, we will just have to accept that we cannot verify a file
  // has not renamed or rotated during restarts - that is, the statefile will only contain the file path

  // Do nothing and return, the vol/idxhi/idxlo FileState entries will be set to 0
}

func is_file_same(path string, info os.FileInfo, state *FileState) bool {
  // Just compare filename
  return (*state.Source == path)
}
