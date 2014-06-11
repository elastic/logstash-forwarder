package system

/* System Documents & Records are maintained by the system.Registrar.
 * This file contains the various types and funcs that provide
 * the functionality.
 */
import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

var NilValue = []byte{}

var E_EXISTING_DOC = errors.New("document exists")

// ----------------------------------------------------------------------------
// Registry
// ----------------------------------------------------------------------------
type registry struct {
	path     string
	rootinfo os.FileInfo
}

// initializes a registry structure.
// if dir is not absolute path, then we use
// current working directory as base path
func openRegistry(dir string) (*registry, error) {

	pwd := ""
	if dir[0] != '/' {
		var e error
		pwd, e = os.Getwd()
		if e != nil {
			return nil, e
		}
	}

	rootpath := path.Join(pwd, dir)

	root, e := os.Open(rootpath)
	if e != nil {
		return nil, e
	}
	info, e := root.Stat()
	if e != nil {
		return nil, e
	}
	if !info.IsDir() {
		return nil, errors.New(rootpath + " is not a directory")
	}

	r := &registry{
		path:     rootpath,
		rootinfo: info,
	}

	return r, nil
}

func (r *registry) updateDocument(doc *document) (bool, error) {
	docpath, docname := docpathForKey(r.path, doc.key)
	//	log.Printf("registry.updateDocument: \n\t%s \n\t%s \n\t%s", doc.key, docname, docpath)
	return updateDocument(doc, path.Join(docpath, docname))
}

func (r *registry) readDocument(key DocId) (*document, error) {
	docpath, docname := docpathForKey(r.path, key)
	//	log.Printf("registry.getDocument: \n\t%s \n\t%s \n\t%s", key, docname, docpath)
	return loadDocument(key, path.Join(docpath, docname))
}

func (r *registry) createDocument(key DocId, data map[string][]byte) (*document, error) {
	docpath, docname := docpathForKey(r.path, key)
	//	log.Printf("registry.getDocument: \n\t%s \n\t%s \n\t%s", key, docname, docpath)
	return newDocument(key, docpath, docname, data)
}

//func confFile(capability string) string {
//	return strings.ToUpper(capability + "-conf")
//}

func docpathForKey(lsfpath string, key DocId) (filepath, filename string) {
	docid := string(key)
	keyparts := strings.Split(docid, ".")
	kplen := len(keyparts)
	switch kplen {
	case 1:
		return path.Join(lsfpath, "."), strings.ToUpper(docid)
	default:
		docname := keyparts[kplen-1]
		filepath = strings.Replace(docid, ".", "/", -1)[:len(docid)-len(docname)]
		return path.Join(lsfpath, filepath), strings.ToUpper(docname)
	}
}

// ----------------------------------------------------------------------------
// Document (k/v)
// ----------------------------------------------------------------------------

type DataMap interface {
	Mappings() map[string][]byte
}
type DocId string

type Document interface {
	Id() DocId
	Keys() []string
	Mappings() map[string][]byte
	Get(key string) []byte
	Set(key string, value []byte) []byte
	Delete(key string) bool
}

type document struct {
	key      DocId
	info     *os.FileInfo
	readtime time.Time
	records  map[string][]byte
	lock     Lock
	dirty    bool
}

func (d *document) Mappings() map[string][]byte {
	if d == nil {
		return map[string][]byte{}
	}
	mappings := make(map[string][]byte, len(d.records))
	for k, v := range d.records {
		mappings[k] = v
	}
	return mappings
}
func (d *document) Keys() []string {
	if d == nil {
		return []string{}
	}
	keys := make([]string, len(d.records))
	n := 0
	for k, _ := range d.records {
		keys[n] = k
		n++
	}
	return keys
}

func (d *document) Id() DocId {
	return d.key
}

func (d *document) Get(k string) []byte {
	prev := d.records[k]
	if prev != nil {
		return prev
	}
	return nil
}

func (d *document) Set(k string, v []byte) []byte {
	prev := d.Get(k)
	d.records[k] = v
	return prev
}

func (d *document) Delete(k string) bool {
	existed := d.records[k] != nil
	delete(d.records, k)
	return existed
}

// acquire resource lock
// create file if not existing.
// write data
// close file
// release lock
func newDocument(dockey DocId, fpath, fname string, data map[string][]byte) (*document, error) {
	//	log.Printf("newDocument: %q %q %q", dockey, fpath, fname)
	dstat, e := os.Stat(fpath)
	if e != nil {
		// REVU: ok to create the directory
		e := os.MkdirAll(fpath, os.ModePerm)
		if e != nil {
			return nil, fmt.Errorf("newDocument: error creating dir %q - %s", fpath, e.Error())
		}
	} else if !dstat.IsDir() {
		panic(fmt.Errorf("BUG - %s expected to be a directory", fpath))
	}

	filename := path.Join(fpath, fname)

	// acquire lock for file
	lock, ok, e := LockResource(filename, "create document "+string(dockey))
	if e != nil {
		return nil, fmt.Errorf("newDocument: error acquiring lock for %q - %s", dockey, e.Error())
	}
	if !ok {
		return nil, fmt.Errorf("error newDocument: file already exists: %q", filename)
	}
	defer lock.Unlock()

	_, e = os.Stat(filename)
	// should not exist
	if !os.IsNotExist(e) {
		return nil, fmt.Errorf("error newDocument: file already exists: %q", filename)
	}

	file, e := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(0644))
	if e != nil {
		return nil, fmt.Errorf("newDocument: error on create %q - %s", filename, e)
	}
	//	log.Println("newDocument: created file %q", file)
	info, _ := file.Stat()
	defer file.Close()

	records := make(map[string][]byte, len(data))
	doc := &document{dockey, &info, time.Now(), records, lock, false}
	for k, v := range data {
		records[k] = v
	}
	e = doc.Write(file)
	if e != nil {
		return nil, e
	}

	return doc, nil
}

func (d *document) encode(k string, v []byte) []byte {
	var buf []byte
	buf = append(buf, k...)
	buf = append(buf, ": "...)
	buf = append(buf, v...)
	buf = append(buf, "\n"...)
	return buf
}

func (d *document) Bytes() []byte {
	var buf []byte
	for k, v := range d.records {
		buf = append(buf, d.encode(k, v)...)
	}
	return buf
}

func (d *document) Write(w io.Writer) error {
	data := d.Bytes()
	n, e := w.Write(data)
	if e != nil {
		return e
	}
	if n < len(data) {
		return fmt.Errorf("error updateDocument: only wrote %d of %d", n, len(data))
	}
	return nil
}

// Saves document: if dirty, dirty flag cleared; otherwise returns false, nil.
// Write Lock acquired for duration (attempted)
// New document file is atomically swapped.
func updateDocument(doc *document, filename string) (bool, error) {
	//	if !doc.dirty {
	//		return false, nil
	//	}

	// create temp file
	swapfile := filename + ".new"
	file, e := os.OpenFile(swapfile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(0644))
	if e != nil {
		return false, fmt.Errorf("updateDocument: error on create %q - %s", swapfile, e)
	}
	defer file.Close()

	e = doc.Write(file)
	if e != nil {
		return false, e
	}

	// acquire lock for doc file
	lock, ok, e := LockResource(filename, "create document "+string(doc.key))
	if e != nil {
		return false, fmt.Errorf("updateDocument: error acquiring lock for %q - %s", doc.key, e.Error())
	}
	if !ok {
		return false, fmt.Errorf("error updateDocument: file already exists: %q", filename)
	}
	defer lock.Unlock()

	if e = os.Remove(filename); e != nil {
		return false, fmt.Errorf("updateDocument: error (swap) removing %q - %s", doc.key, e.Error())
	}
	if e = os.Rename(swapfile, filename); e != nil {
		return false, fmt.Errorf("updateDocument: error (swap) renaming %q - %s", doc.key, e.Error())
	}
	log.Println("updateDocument: updated file %q", filename)

	return true, nil
}

// load for read.
// read file and closes it.
// REVU TODO what if locked?
func loadDocument(dockey DocId, filename string) (*document, error) {

	// verify document file
	info, e := os.Stat(filename)
	if e != nil {
		return nil, e
	}
	if info.IsDir() {
		return nil, errors.New(filename + " is not a file")
	}
	// REVU: lock checks could go here.

	// open and defer close document file
	file, e := os.Open(filename)
	if e != nil {
		return nil, fmt.Errorf("error loadDocument: os.Open %s - %s", filename, e.Error())
	}
	defer file.Close()

	// read document file
	bufsize := int(info.Size())
	buf := make([]byte, bufsize)
	n, e := file.Read(buf)
	if e != nil {
		return nil, fmt.Errorf("error loadDocument: Read %s - %s", filename, e.Error())
	}
	if n < bufsize {
		return nil, fmt.Errorf("error loadDocument: Read %s - only read %d of %d", filename, n, bufsize)
	}

	// create and load document
	doc := &document{dockey, &info, time.Now(), make(map[string][]byte), nil, false}
	lines := strings.Split(string(buf), "\n")
	for _, line := range lines {
		if len(line) > 0 && line[0] != '#' {
			//			log.Printf("%s\n", line)
			tuple2 := strings.SplitN(line, ":", 2)
			if len(tuple2) != 2 {
				return nil, fmt.Errorf("%q corrupted. malformed record: %q", line)
			}
			// trim all whitespace from key and value
			tuple2[0] = strings.Trim(tuple2[0], "\t ")
			tuple2[1] = strings.Trim(tuple2[1], "\t ")
			doc.records[tuple2[0]] = []byte(tuple2[1])
		}
	}

	//	log.Println("newDocument: done")
	return doc, nil
}

// ----------------------------------------------------------------------------
// Record
// ----------------------------------------------------------------------------

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
		log.Printf("ERROR - %s", e.Error())
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
