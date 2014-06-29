package system

// registrar.go: implementation of lsf/system.Registrar

import (
	"fmt"
	"lsf/system/process"
)

func StartRegistry(basepath string) (Registrar, error) {
	registrar, e := newRegistrar(basepath)
	if e != nil {
		return nil, e
	}
	// start the registrar active component
	go registrar.run()

	return registrar, nil
}

// Launches a goroutine to process user requests from a request queue.
// This mechanism provides the in-memory linearlization of access to the
// shared system resources managed by system, per semantics of system.Registrar
func (r *registrar) run() {
	defer func() {
		// REVU: request.execute() returns errors via channels
		//       and wraps calls to lsf/lsfun functions (which
		//       are not to panic(?)). << TODO: affirm
	}()

	for {
		select {
		case request := <-r.ui:
			request.result <- request.execute()
		case <-r.Command():
			r.Report() <- stat{nil, NilValue}
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
	*process.Control
	reg *registry
	ui  chan req
}

func newRegistrar(basepath string) (*registrar, error) {
	registry, e := openRegistry(basepath)
	if e != nil {
		return nil, e
	}

	regisrar := &registrar{
		process.NewProcessControl(),
		registry,
		make(chan req, 12),
	}
	return regisrar, nil
}

// ----------------------------------------------------------------------------
// interface: Registrar
// ----------------------------------------------------------------------------

func (r *registrar) String() string {
	s := fmt.Sprintf("registrar: path %s", r.reg.path)
	return s
}

func (r *registrar) DeleteDocument(key DocId) (bool, error) {
	fn := func() interface{} {
		ok, e := r.reg.deleteDocument(key)
		if e != nil {
			return e
		}
		return ok
	}
	return r.dispatch1(fn)
}

func (r *registrar) UpdateDocument(doc Document) (bool, error) {
	fn := func() interface{} {
		ok, e := r.reg.updateDocument(doc.(*document))
		if e != nil {
			return e
		}
		return ok
	}
	return r.dispatch1(fn)
}
func (r *registrar) ReadDocument(key DocId) (Document, error) {
	fn := func() interface{} {
		doc, e := r.reg.readDocument(key)
		if e != nil {
			return e
		}
		return doc
	}
	return r.dispatch0(fn)
}

func (r *registrar) CreateDocument(key DocId, data map[string][]byte) (Document, error) {
	fn := func() interface{} {
		doc, e := r.reg.createDocument(key, data)
		if e != nil {
			return e
		}
		return doc
	}
	return r.dispatch0(fn)
}

func (r *registrar) dispatch0(fn func() interface{}) (Document, error) {
	resch := makeResChan()
	r.ui <- req{resch, fn}
	result := <-resch
	return mapDocResult(result)
}

func (r *registrar) dispatch1(fn func() interface{}) (bool, error) {
	resch := makeResChan()
	r.ui <- req{resch, fn}
	result := <-resch
	return mapBoolResult(result)
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
