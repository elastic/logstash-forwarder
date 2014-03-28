package main

import (
  "os"
  "syscall"
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

func is_filestate_same(path string, info os.FileInfo, state *FileState) bool {
  // Just compare filename
  return (*state.Source == path)
}

func open_file_no_lock(path string) (*os.File, error) {
  // We will call CreateFile directly so we can pass in FILE_SHARE_DELETE
  // This ensures that a program can still rotate the file even though we have it open
  pathp, err := syscall.UTF16PtrFromString(path)
  if err != nil {
    return nil, err
  }

  var sa *syscall.SecurityAttributes

  handle, err := syscall.CreateFile(
    pathp, syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
    sa, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
  if err != nil {
    return nil, err
  }

  return os.NewFile(uintptr(handle), path), nil
}
