package main

import (
  "os"
  "syscall"
)

func file_ids(info *os.FileInfo) (uint64, uint64) {
  fstat := (*info).Sys().(*syscall.Stat_t)
  return fstat.Ino, fstat.Dev
}
