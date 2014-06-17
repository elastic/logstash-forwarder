package schema

import (
	"fmt"
	"lsf/system"
)

// All streams have a unique identity.
// The simple identity 'StreamId' is unique in context of the stream's home port.
// The globally unique Stream Identifier is PortHome/StreamId
type StreamId string

const AnonStreamId StreamId = "" // TODO REVU

// ----------------------------------------------------------------------
// LogStream
// ----------------------------------------------------------------------

type LogStream struct {
	// Unique (in context of Port/Env) identifier of a stream
	Id StreamId
	// Path to the log files
	Path string
	// JournalModel
	//	JournalModel JournalModel
	JournalModel journalModel
	// Log filename pattern
	Pattern string
	// Stream's semantic structure
	Fields map[string]string // lazy

	// one mapping entry per lsf command e.g.
	// Records["track"] // maps to tracking logs for stream
	records map[string]*LogRecord // lazy
}

var DefaultStreamMappings = defmapping{
	"journal-model": []byte(JournalModel.Rotation),
}

// REVU: TODO sort mappings at sysrec..
func (t *LogStream) Mappings() map[string][]byte {
	m := make(map[string][]byte)
	m["id"] = []byte(t.Id)
	m["file-path"] = []byte(t.Path)
	m["name-pattern"] = []byte(t.Pattern)
	m["journal-model"] = []byte(t.JournalModel)
	return m
}

func (t *LogStream) String() string {
	return fmt.Sprintf("logstream %s %s %s %s %s", t.Id, t.Path, t.JournalModel, t.Pattern, t.Fields)
}

func LogStreamDigest(doc system.Document) string {
	logstream := DecodeLogStream(doc)
	return logstream.String()
}

func DecodeLogStream(data system.DataMap) *LogStream {
	m := data.Mappings()
	return &LogStream{
		Id:           StreamId(string(m["id"])),
		Path:         string(m["file-path"]),
		JournalModel: journalModel(string(m["journal-model"])),
		Pattern:      string(m["name-pattern"]),
		Fields:       make(map[string]string), // TODO: fields needs a solution
		records:      make(map[string]*LogRecord),
	}
}

func NewLogStream(id StreamId, path string, journalModel journalModel, namingPattern string, fields map[string]string) *LogStream {
	return &LogStream{
		Id:           id,
		Path:         path,
		JournalModel: journalModel,
		Pattern:      namingPattern,
		Fields:       fields,
		records:      make(map[string]*LogRecord),
	}
}
