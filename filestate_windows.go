package main

import (
  "os"
  "syscall"
  "reflect"
  "log"
)

type FileState struct {
  Source *string `json:"source,omitempty"`
  Offset int64   `json:"offset,omitempty"`
  Vol    uint32  `json:"vol,omitempty"`
  IdxHi  uint32  `json:"idxhi,omitempty"`
  IdxLo  uint32  `json:"idxlo,omitempty"`
}

func file_ids(info *os.FileInfo, state *FileState) {
  // For information on the following, see Go source: src/pkg/os/types_windows.go
  // This is the only way we can get at the idxhi and idxlo
  // Unix it is much easier as syscall.Stat_t is exposed and os.FileInfo interface has a Sys() method to get a syscall.Stat_t
  // Unfortunately, the relevant Windows information is in a private struct so we have to dig inside

  // NOTE: This WILL be prone to break if Go source changes, but I'd rather just fix it if it does or make it fail gracefully

  // info is *os.FileInfo which is a pointer to a
  // - os.FileInfo interface of a
  // - *os.fileStat (holding methods) which is a pointer to a
  // - os.fileStat (holding data)

  // Ensure that the numbers are loaded by calling os.SameFile
  // os.SameFile will call sameFile (types_windows.go) which will call *os.fileStat's loadFileId
  // Reflection panics if we try to call an unexpected method; but anyway this is much safer and more reliable
  os.SameFile(*info, *info)

  // If any of the following fails, report the library has changed and recover and return 0s
  defer func() {
    if r := recover(); r != nil {
      log.Printf("WARNING: File rotations that occur while LogStash Forwarder is not running will NOT be detected due to an incompatible change to the Go library used for compiling. This is a bug, please report it.\n")
      state.Vol = 0
      state.IdxHi = 0
      state.IdxLo = 0
    }
  }()

  // Following makes fstat hold os.fileStat
  fstat := reflect.ValueOf(info).Elem().Elem().Elem()

  // To get the data, we need the os.fileStat that fstat points to, so one more Elem()
  state.Vol = uint32(fstat.FieldByName("vol").Uint())
  state.IdxHi = uint32(fstat.FieldByName("idxhi").Uint())
  state.IdxLo = uint32(fstat.FieldByName("idxlo").Uint())
}

func is_filestate_same(path string, info os.FileInfo, state *FileState) bool {
  istate := &FileState{}
  file_ids(&info, istate)
  return (istate.Vol == state.Vol && istate.IdxHi == state.IdxHi && istate.IdxLo == state.IdxLo)
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
