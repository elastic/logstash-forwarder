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

// REVU: do we (really) need this?

// ProcControl provides system process supervisors the means to
// interact with the managed process.
type Process interface {
	// Returns Signal channel to the process
	Signal() chan<- interface{}
	// Returns Stat channel from the process
	Status() <-chan interface{}
}

// REVU: This contortion is to have a bipolar view
// on the 2 one-way channels between a supervisor and process
// The ProcessSpi is
type Supervisor interface {
	// Returns Signal channel to the process
	Command() <-chan interface{}
	// Returns Stat channel from the process
	Report() chan<- interface{}
}
