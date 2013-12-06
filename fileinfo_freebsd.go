package main

import (
  "os"
  "syscall"
)

func file_ids(info *os.FileInfo) (uint32, uint32) {
  fstat := (*(info)).Sys().(*syscall.Stat_t)
  return fstat.Ino, fstat.Dev
}
