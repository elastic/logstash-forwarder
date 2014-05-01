package main

import (
  "os"
)

func lookup_file_ids(file string, info os.FileInfo, fileinfo map[string]ProspectorInfo, missingfiles map[string]os.FileInfo) string {
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

func lookup_file_ids_resume(file string, info os.FileInfo, initial map[string]*FileState) string {
  for kf, ki := range initial {
    if kf == file {
      continue
    }
    if is_filestate_same(file, info, ki) {
      return kf
    }
  }

  return ""
}
