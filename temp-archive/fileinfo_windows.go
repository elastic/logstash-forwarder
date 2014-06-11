package main

import (
  "os"
)

func file_ids(info *os.FileInfo) (uint64, uint64) {
  // No dev and inode numbers on windows, right?
  return 0, 0
}
