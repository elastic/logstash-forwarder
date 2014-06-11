package main

import (
  "os"
  "syscall"
)

func file_ids(info *os.FileInfo) (uint64, int32) {
  fstat := (*(info)).Sys().(*syscall.Stat_t)
  return fstat.Ino, fstat.Dev
}
