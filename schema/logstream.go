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

// recorded elements of LogStream object
var logStreamElem = struct {
	id, basepath, pattern, model string
}{
	id:       "id",
	basepath: "basepath",
	pattern:  "name-pattern",
	model:    "journal-model",
}

var DefaultStreamMappings = defmapping{
	logStreamElem.model: []byte(JournalModel.Rotation),
}

// REVU: TODO sort mappings at sysrec..
func (t *LogStream) Mappings() map[string][]byte {
	m := make(map[string][]byte)
	m[logStreamElem.id] = []byte(t.Id)
	m[logStreamElem.basepath] = []byte(t.Path)
	m[logStreamElem.pattern] = []byte(t.Pattern)
	m[logStreamElem.model] = []byte(t.JournalModel)
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
		Id:           StreamId(string(m[logStreamElem.id])),
		Path:         string(m[logStreamElem.basepath]),
		JournalModel: journalModel(string(m[logStreamElem.model])),
		Pattern:      string(m[logStreamElem.pattern]),
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
