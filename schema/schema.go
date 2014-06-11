package schema

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lsf/system"
	"net/url"
	"os"
	"time"
)

type defmapping map[string][]byte

func (data defmapping) Mappings() map[string][]byte {
	return data
}

var DefaultSystemMappings = defmapping{}

// ----------------------------------------------------------------------
// Config
// ----------------------------------------------------------------------

// Value struct semantics.
type Config struct {
	Remotes map[PortId]Port
	Streams map[StreamId]LogStream
}

func NewConfig() *Config {
	return &Config{
		Remotes: make(map[PortId]Port),
		Streams: make(map[StreamId]LogStream),
	}
}

// Cover the json bits - we may change our mind about format
func (c *Config) encode() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// Cover the json bits - we may change our mind about format
func decodeConfig(b []byte) (*Config, error) {
	c := &Config{}
	e := json.Unmarshal(b, c)
	if e != nil {
		return nil, e
	}
	return c, nil
}

func WriteConfig(config *Config, fname string) error {
	tempfile := fmt.Sprintf("%s.new", fname)

	confile, e := os.Create(tempfile)
	if e != nil {
		return fmt.Errorf("could not create file %s: %s", tempfile, e)
	}

	buf, e := config.encode()
	if e != nil {
		return fmt.Errorf("error on config.Encode(): %s", e)
	}
	buf = append(buf, byte('\n'))

	n, e := confile.Write(buf)
	if e != nil { // implies n < len(buf) per os.File#Write() docs
		return fmt.Errorf("error on writing config file (%d writen): %s", n, e)
	}

	e = os.Rename(tempfile, fname)
	if e != nil {
		return fmt.Errorf("error renaming new configfile: %s", e)
	}

	fmt.Printf("DEBUG - WROTE %s", *config)
	return nil
}

func readConfig(fname string) (*Config, error) {
	file, e := os.Open(fname)
	if e != nil {
		panic(e)
	}
	defer file.Close()

	buf, e := ioutil.ReadAll(file)
	if e != nil {
		return nil, e
	}
	config, e := decodeConfig(buf)
	if e != nil {
		return nil, e
	}
	return config, nil
}

// ----------------------------------------------------------------------
// Identifiers
// ----------------------------------------------------------------------

// All streams have a unique identity.
// The simple identity 'StreamId' is unique in context of the stream's home port.
// The globally unique Stream Identifier is PortHome/StreamId
type StreamId string

const AnonStreamId StreamId = "" // TODO REVU

type PortId string
type portType int

const (
	localPort  portType = 0
	remotePort          = 1
)

const AnonPortId PortId = ""

// ----------------------------------------------------------------------
// Port
// ----------------------------------------------------------------------

// lsf.RemotePort describes a remote LSF port.
type Port struct {
	//	Id      PortId // TODO REVU first ..
	Address *url.URL
	// todo certs ..
}

func (p Port) Path() string { return p.Address.Path }

// returns nil on "" path
func NewLocalPort(path string) *Port {
	if path == "" {
		return nil
	}
	//	var address *url.URL
	address, e := url.Parse(path)
	if e != nil {
		panic(e) // unexpected
	}

	return &Port{
		//		Id: HexShaDigest(path),
		Address: address,
	}

}

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

type LogStream struct {
	// Unique (in context of Port/Env) identifier of a stream
	Id StreamId
	// Path to the log files
	Path string
	// JournalModel
	JournalModel JournalModel
	// Log filename pattern
	Pattern string
	// Stream's semantic structure
	Fields map[string]string

	// one mapping entry per lsf command e.g.
	// Records["track"] // maps to tracking logs for stream
	records map[string]*LogRecord
}

var DefaultStreamMappings = defmapping{
	"journal-model": []byte(Rotation),
}

// REVU: TODO sort mappings at sysrec..
func (t *LogStream) Mappings() map[string][]byte {
	m := make(map[string][]byte)
	m["id"] = []byte(t.Id)
	m["file-path"] = []byte(t.Path)
	m["name-pattern"] = []byte(t.Pattern)
	m["journal-mode"] = []byte(t.JournalModel)
	return m
}

func (t *LogStream) String() string {
	return fmt.Sprintf("logstream %s %s %s %s %s", t.Id, t.Path, t.JournalModel, t.Pattern, t.Fields)
}

func DecodeLogStream(data system.DataMap) *LogStream {
	m := data.Mappings()
	return &LogStream{
		Id:           StreamId(string(m["id"])),
		Path:         string(m["file-path"]),
		JournalModel: JournalModel(string(m["name-pattern"])),
		Pattern:      string(m["journal-model"]),
		Fields:       make(map[string]string), // TODO: fields needs a solution
		records:      make(map[string]*LogRecord),
	}
}

func NewLogStream(id StreamId, path string, journalModel JournalModel, namingPattern string, fields map[string]string) *LogStream {
	return &LogStream{
		Id:           id,
		Path:         path,
		JournalModel: journalModel,
		Pattern:      namingPattern,
		Fields:       fields,
		records:      make(map[string]*LogRecord),
	}
}

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
