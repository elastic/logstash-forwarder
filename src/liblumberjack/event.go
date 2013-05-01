package liblumberjack

import "os"

type FileEvent struct {
  Source *string `json:"source,omitempty"`
  Offset uint64 `json:"offset,omitempty"`
  Line uint64 `json:"line,omitempty"`
  Text *string `json:"text,omitempty"`

  fileinfo *os.FileInfo
}

type FileState struct {
  Source *string `json:"source,omitempty"`
  Offset uint64 `json:"offset,omitempty"`
  Inode uint64 `json:"inode,omitempty"`
  Device uint64 `json:"device,omitempty"`
}
