package main

type FileState struct {
  Source *string `json:"source,omitempty"`
  Offset int64   `json:"offset,omitempty"`
  Inode  uint64  `json:"inode,omitempty"`
  Device int32   `json:"device,omitempty"`
}
