package system

// registrar.go: implementation of lsf/system.Registrar

import (
	"fmt"
)

func StartRegistry(path string) (Registrar, error) {
	r, e := openRegistry(path)
	if e != nil {
		return nil, e
	}
	ui := make(chan req, 12) 			// REVU: magic number.. aim is to allow fast enqueue of requests.
	cancel := make(chan struct{}, 1)
	done := make(chan stat, 1)

	registrar := &registrar{r, ui, done, cancel}
	go beTheRegistrar(r, ui, cancel, done)

	return registrar, nil
}

// Launches a goroutine to process user requests from a request queue.
// This mechanism provides the in-memory linearlization of access to the
// shared system resources managed by system, per semantics of system.Registrar
func beTheRegistrar(r *registry, ui <-chan req, cancel <-chan struct{}, done chan<- stat) {
	defer func() {
		// REVU: request.execute() returns errors via channels
		//       and wraps calls to lsf/lsfun functions (which
		//       are not to panic(?)). << TODO: affirm
	}()

	for {
		select {
		case request := <-ui:
			// REVU: TODO:
			request.result <- request.execute()
		case <-cancel:
			done <- stat{nil, NilValue}
			return
		}
	}
}


// ----------------------------------------------------------------------------
// registrar
// ----------------------------------------------------------------------------

// registrar implements system doc registrar functionality and supports the
// lsf/system.Registrar interface.
type registrar struct {
	reg    *registry
	ui     chan req
	done   chan stat
	cancel chan struct{}
}


// ----------------------------------------------------------------------------
// interface: Registrar
// ----------------------------------------------------------------------------

func (r *registrar) String() string {
	s := fmt.Sprintf("registrar: path %s", r.reg.path)
	return s
}
func (r *registrar) Done() <-chan stat     { return r.done }
func (r *registrar) Stop() chan<- struct{} { return r.cancel }

func (r *registrar) DeleteDocument(key DocId) (bool, error) {
	resch := makeResChan()
	fn := func() interface{} {
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

// ----------------------------------------------------------------------------
// concurrent async request dispatch mechanism
// ----------------------------------------------------------------------------

type stat struct {
	err error
	dat []byte
}

// query type just wraps the delegated registrar func invokes
type query func() interface{}

// an async request is a tuple wrapping result callback channel
// and the actual (query) func invoke
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
