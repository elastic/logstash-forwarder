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
	stream    string
	command   string
	Timestamp time.Time
}

// ----------------------------------------------------------------------
// JournalModel
// ----------------------------------------------------------------------

type journalModel string

// enum
var JournalModel = struct {
	Rotation, Rollover journalModel
}{
	Rotation: "rotation",
	Rollover: "rollover",
}

func ToJournalModel(v string) journalModel {
	switch journalModel(v) {
	case JournalModel.Rotation:
		return JournalModel.Rotation
	case JournalModel.Rollover:
		return JournalModel.Rollover
	}
	return journalModel("")
}

// ----------------------------------------------------------------------
// Identifiers
// ----------------------------------------------------------------------

// The Stash - universal journal schema for log streams
type LogJournal struct {
	Stream  string
	Entries []*LogJournalEntry
}

// An entry in the LogJournal
type LogJournalEntry struct {
	Timestamp time.Time
	Data      []byte // REVU: TODO: consider the string..
}
