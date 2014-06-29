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
	// Log filename Pattern
	Pattern string
	// Stream's semantic structure
	Fields map[string]string // lazy

	// one mapping entry per lsf command e.g.
	// Records["track"] // maps to tracking logs for stream
	records map[string]*LogRecord // lazy
}

// recorded elements of LogStream object
var LogStreamElem = struct {
	Id, BasePath, Pattern, JournalModel string
}{
	Id:           "id",
	BasePath:     "basepath",
	Pattern:      "name-pattern",
	JournalModel: "journal-model",
}

var DefaultStreamMappings = defmapping{
	LogStreamElem.JournalModel: []byte(JournalModel.Rotation),
}

// REVU: TODO sort mappings at sysrec..
func (t *LogStream) Mappings() map[string][]byte {
	m := make(map[string][]byte)
	m[LogStreamElem.Id] = []byte(t.Id)
	m[LogStreamElem.BasePath] = []byte(t.Path)
	m[LogStreamElem.Pattern] = []byte(t.Pattern)
	m[LogStreamElem.JournalModel] = []byte(t.JournalModel)
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
		Id:           StreamId(string(m[LogStreamElem.Id])),
		Path:         string(m[LogStreamElem.BasePath]),
		JournalModel: journalModel(string(m[LogStreamElem.JournalModel])),
		Pattern:      string(m[LogStreamElem.Pattern]),
		Fields:       make(map[string]string), // REVU: an array of tags TODO mod []string ..
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
