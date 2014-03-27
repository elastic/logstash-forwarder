package main

import (
  "os"
  "syscall"
)

type FileState struct {
  Source *string `json:"source,omitempty"`
  Offset int64   `json:"offset,omitempty"`
  Inode  uint64  `json:"inode,omitempty"`
  Device int32   `json:"device,omitempty"`
}

func file_ids(info *os.FileInfo, state *FileState) {
  fstat := (*info).Sys().(*syscall.Stat_t)
  state.Inode = fstat.Ino
  state.Device = fstat.Dev
}

func is_filestate_same(path string, info os.FileInfo, state *FileState) bool {
  istate := &FileState{}
  file_ids(&info, istate)
  return (istate.Inode == state.Inode && istate.Device == state.Device)
}

func open_file_no_lock(path string) (*os.File, error) {
  return os.Open(path)
}
