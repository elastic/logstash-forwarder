package main

type FileState struct {
  Source *string `json:"source,omitempty"`
  Offset int64 `json:"offset,omitempty"`
  Inode uint32 `json:"inode,omitempty"`
  Device uint32 `json:"device,omitempty"`
}
