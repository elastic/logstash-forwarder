package system

// ----------------------------------------------------------------------------
// System Document Registrar
// ----------------------------------------------------------------------------

// Registrar defines the semantics of accessing and manipulating (lsf/system)
// system documents.
type Registrar interface {
	// Reads the document and returns snapshot value.
	// No locks held. No files open
	ReadDocument(DocId) (Document, error)
	// Creates new document with given map (of record data).
	// Returns the document snapshot (per ReadDocument)
	// No locks head. No files open
	CreateDocument(DocId, map[string][]byte) (Document, error)
	// Saves document (if dirty) - dirty flag cleared; otherwise NOP.
	// Write Lock acquired for duration (attempted)
	// New document file is atomically swapped.
	UpdateDocument(Document) (bool, error)
	//
	DeleteDocument(DocId) (bool, error)
	// stop.
	// release all resources.
	Stop() chan<- struct{}
	// signals Registrar stopped.
	// signals all resources released.
	Done() <-chan stat
	// identity info & status
	String() string
}

// ----------------------------------------------------------------------------
// System Process Control
// ----------------------------------------------------------------------------

// ProcControl provides system process supervisors the means to
// interact with the managed process.
type ProcControl interface {
	// Returns Signal channel to the process
	Sig() chan<- interface{}
	// Returns Stat channel from the process
	Stat() <-chan interface{}
}


