package liblumberjack

type FileEvent struct {
  Source *string `json:"source,omitempty"`
  Offset *uint64 `json:"offset,omitempty"`
  Line *uint64 `json:"line,omitempty"`
  Text *string `json:"text,omitempty"`
}
