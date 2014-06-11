package main

import (
  "os"
)

func is_file_same(path string, info os.FileInfo, state *FileState) bool {
  // Do we have any other way to validate a file is the same file
  // under windows?
  return path == *state.Source
}

func is_fileinfo_same(a os.FileInfo, b os.FileInfo) bool {
  // Anything meaningful to compare on file infos?
  return true
}

func is_file_renamed(file string, info os.FileInfo, fileinfo map[string]os.FileInfo) bool {
  // Can we detect if a file was renamed on Windows?
  return false
}
