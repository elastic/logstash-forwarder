package schema

import (
	"time"
)

// defines the generic logical form of all schema entities
type defmapping map[string][]byte

// all schema entities support this interface.
type Encoder interface {
	Mappings() map[string][]byte
	String() string
	Bytes() []byte
}
type Decoder interface {
	Map(mapping defmapping) interface{}
}

func (data defmapping) Mappings() map[string][]byte {
	return data
}

var DefaultSystemMappings = defmapping{}

// ----------------------------------------------------------------------
// LogRecord
// ----------------------------------------------------------------------

type LogRecord struct {
	stream    StreamId
	command   string
	Timestamp time.Time
}

// ----------------------------------------------------------------------
// JournalModel
// ----------------------------------------------------------------------

type JournalModel string

const (
	Rotation JournalModel = "rotation" // file sequences
	Rollover              = "rollover" // truncated
)

// ----------------------------------------------------------------------
// Identifiers
// ----------------------------------------------------------------------

// The Stash - universal journal schema for log streams
type LogJournal struct {
	Stream  StreamId
	Entries []*LogJournalEntry
}

// An entry in the LogJournal
type LogJournalEntry struct {
	Timestamp time.Time
}
