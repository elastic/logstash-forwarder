package main

type FileState struct {
  Source *string `json:"source,omitempty"`
  Offset int64 `json:"offset,omitempty"`
  Vol   uint64 `json:"vol,omitempty"`
  Idxhi uint64 `json:"idxhi,omitempty"`
  Idxlo uint64 `json:"idxlo,omitempty"`
}
