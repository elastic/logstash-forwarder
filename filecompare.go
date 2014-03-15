// +build !windows

package main

import (
  "os"
  "syscall"
)

func is_file_same(path string, info os.FileInfo, state *FileState) bool {
  fstat := info.Sys().(*syscall.Stat_t)
  return (fstat.Ino == state.Inode && fstat.Dev == state.Device)
}

func is_fileinfo_same(a os.FileInfo, b os.FileInfo) bool {
  af := a.Sys().(*syscall.Stat_t)
  bf := b.Sys().(*syscall.Stat_t)
  return (af.Dev == bf.Dev && af.Ino == bf.Ino)
}

func is_file_renamed(file string, info os.FileInfo, fileinfo map[string]ProspectorInfo, missingfiles map[string]os.FileInfo) string {
  // NOTE(driskell): What about using golang's func os.SameFile(fi1, fi2 FileInfo) bool instead?
  stat := info.Sys().(*syscall.Stat_t)

  for kf, ki := range fileinfo {
    if kf == file {
      continue
    }
    ks := ki.fileinfo.Sys().(*syscall.Stat_t)
    if stat.Dev == ks.Dev && stat.Ino == ks.Ino {
      return kf
    }
  }

  // Now check the missingfiles
  for kf, ki := range missingfiles {
    ks := ki.Sys().(*syscall.Stat_t)
    if stat.Dev == ks.Dev && stat.Ino == ks.Ino {
      return kf
    }
  }
  return ""
}

func is_file_renamed_resumelist(file string, info os.FileInfo, initial map[string]*FileState) string {
  // NOTE(driskell): What about using golang's func os.SameFile(fi1, fi2 FileInfo) bool instead?
  stat := info.Sys().(*syscall.Stat_t)

  for kf, ki := range initial {
    if kf == file {
      continue
    }
    if stat.Dev == ki.Device && stat.Ino == ki.Inode {
      return kf
    }
  }

  return ""
}
