package system

/* System Documents & Records are maintained by the system.Registrar.
 * This file contains the various types and funcs that provide
 * the functionality.
 */
import (
	"fmt"
)

// ----------------------------------------------------------------------------
// Registrar
// ----------------------------------------------------------------------------

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

func StartRegistry(path string) (Registrar, error) {
	r, e := openRegistry(path)
	if e != nil {
		return nil, e
	}
	ui := make(chan req, 12)
	cancel := make(chan struct{}, 1)
	done := make(chan stat, 1)

	registrar := &registrar{r, ui, done, cancel}
	go beTheRegistrar(r, ui, cancel, done)

	return registrar, nil
}

type registrar struct {
	reg    *registry
	ui     chan req
	done   chan stat
	cancel chan struct{}
}

func (r *registrar) String() string {
	s := fmt.Sprintf("registrar: path %s", r.reg.path)
	return s
}
func (r *registrar) Done() <-chan stat     { return r.done }
func (r *registrar) Stop() chan<- struct{} { return r.cancel }

func (r *registrar) DeleteDocument(key DocId) (bool, error) {
	resch := makeResChan()
	fn := func() interface{} {
		//		log.Printf("func: get document %s", string(key))
		ok, e := r.reg.deleteDocument(key)
		if e != nil {
			return e
		}
		return ok
	}
	r.ui <- req{resch, fn}
	result := <-resch
	return mapBoolResult(result)
}

func (r *registrar) UpdateDocument(doc Document) (bool, error) {
	resch := makeResChan()
	fn := func() interface{} {
		//		log.Printf("func: update document %s", string(doc.Id()))
		ok, e := r.reg.updateDocument(doc.(*document))
		if e != nil {
			return e
		}
		return ok
	}
	r.ui <- req{resch, fn}
	result := <-resch
	return mapBoolResult(result)
}
func (r *registrar) ReadDocument(key DocId) (Document, error) {
	resch := makeResChan()
	fn := func() interface{} {
		//		log.Printf("func: get document %s", string(key))
		doc, e := r.reg.readDocument(key)
		if e != nil {
			return e
		}
		return doc
	}
	r.ui <- req{resch, fn}
	result := <-resch
	return mapDocResult(result)
}

func (r *registrar) CreateDocument(key DocId, data map[string][]byte) (Document, error) {
	resch := makeResChan()
	fn := func() interface{} {
		//		log.Println("func: create document: %q", string(key))
		doc, e := r.reg.createDocument(key, data)
		if e != nil {
			return e
		}
		return doc
	}
	r.ui <- req{resch, fn}
	result := <-resch
	return mapDocResult(result)
}

type stat struct {
	err error
	dat []byte
}

type query func() interface{}

type req struct {
	result  chan<- interface{}
	execute query
}

func makeResChan() chan interface{} { return make(chan interface{}, 1) }

func mapDocResult(result interface{}) (Document, error) {
	switch t := result.(type) {
	case Document:
		return t, nil
	case error:
		return nil, t
	default:
		panic("BUG - unexpected type value")
	}
}
func mapBoolResult(result interface{}) (bool, error) {
	switch t := result.(type) {
	case bool:
		return t, nil
	case error:
		return false, t
	default:
		panic("BUG - unexpected type value")
	}
}

func beTheRegistrar(r *registry, ui <-chan req, cancel <-chan struct{}, done chan<- stat) {
	defer func() {}()

	for {
		select {
		case request := <-ui:
			// process request
			//			log.Printf("@request: Registrar: process request %s", request)
			request.result <- request.execute()
		case <-cancel:
			//			log.Println("@cancel: Registrar: stopping")
			done <- stat{nil, NilValue}
			return
		}
	}
}
