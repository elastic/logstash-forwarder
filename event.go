package main

import "os"

type FileEvent struct {
  Source *string `json:"source,omitempty"`
  Offset int64 `json:"offset,omitempty"`
  Line uint64 `json:"line,omitempty"`
  Text *string `json:"text,omitempty"`
  Fields *map[string]string

  fileinfo *os.FileInfo
}

type FileState struct {
  Source *string `json:"source,omitempty"`
  Offset int64 `json:"offset,omitempty"`
  Inode uint64 `json:"inode,omitempty"`
  Device uint64 `json:"device,omitempty"`
}
