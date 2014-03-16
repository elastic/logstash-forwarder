package main

import (
  "os"
)

func is_file_renamed(file string, info os.FileInfo, fileinfo map[string]ProspectorInfo, missingfiles map[string]os.FileInfo) string {
  for kf, ki := range fileinfo {
    if kf == file {
      continue
    }
    if os.SameFile(ki.fileinfo, info) {
      return kf
    }
  }

  // Now check the missingfiles
  for kf, ki := range missingfiles {
    if os.SameFile(info, ki) {
      return kf
    }
  }
  return ""
}

func is_file_renamed_resumelist(file string, info os.FileInfo, initial map[string]*FileState) string {
  for kf, ki := range initial {
    if kf == file {
      continue
    }
    if is_file_same(file, info, ki) {
      return kf
    }
  }

  return ""
}
