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

func is_file_renamed(file string, info os.FileInfo, fileinfo map[string]ProspectorInfo, missingfiles map[string]os.FileInfo) string {
  // Can we detect if a file was renamed on Windows?
  // NOTE(driskell): What about using golang's func os.SameFile(fi1, fi2 FileInfo) bool?
  return ""
}

func is_file_renamed_resumelist(file string, info os.FileInfo, initial map[string]*FileState) string {
  // Can we detect if a file was renamed on Windows?
  // NOTE(driskell): What about using golang's func os.SameFile(fi1, fi2 FileInfo) bool?
  return ""
}
