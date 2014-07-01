package system

import (
	"bytes"
	"fmt"
	"io"
	"lsf/panics"
	"os"
	"strings"
	"time"
)

var E_EXISTING_DOC = fmt.Errorf("document exists")

// ----------------------------------------------------------------------------
// Document (k/v)
// ----------------------------------------------------------------------------

type DataMap interface {
	Mappings() map[string][]byte
}

// Document represents a flat document model of a set of named records,
// i.e. a basic k/v container.
type Document interface {
	// Return the document id
	Id() string
	// Return the document key set
	Keys() []string
	// Return the document data
	Mappings() map[string][]byte
	// Get a specific record by key
	Get(key string) []byte
	// Set a specific record by key.
	// Returns previous record value (if any)
	Set(key string, value []byte) []byte
	// Updates the document records.
	// Returns the previous mappings, which may be an empty map
	// if documentn is not changed.
	SetAll(data map[string][]byte) map[string][]byte
	// Deletes a record.
	// Returns true if record existed.
	Delete(key string) bool
	// Returns true if document has been modified after load
	IsDirty() bool
}

type document struct {
	key      string
	info     *os.FileInfo // REVU: use fs.Object instead?
	readtime time.Time
	records  map[string][]byte
	lock     Lock
	dirty    bool
}

func (d *document) IsDirty() bool {
	// impl. note:
	// dirty flag to be set *only* in Set() and SetAll()
	return d.dirty
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

func (d *document) Id() string {
	return d.key
}

func (d *document) Get(k string) []byte {
	return d.records[k]
}

func (d *document) Set(k string, v []byte) []byte {
	prev := d.records[k]
	if bytes.Compare(prev, v) != 0 {
		d.dirty = true
		d.records[k] = v
	}
	return prev
}

func (d *document) SetAll(data map[string][]byte) (previous map[string][]byte) {
	previous = make(map[string][]byte, len(data))
	for k, v := range data {
		prev := d.records[k]
		// yes, we repeat ourself here (c.f. doc.Set) but it is more efficient
		if bytes.Compare(prev, v) != 0 {
			d.dirty = true
			d.records[k] = v
			previous[k] = prev
		}
	}
	return
}

func (d *document) Delete(k string) bool {
	existed := d.records[k] != nil
	delete(d.records, k)
	return existed
}

type DocumentDigestFn func(Document) string

// acquire resource lock
// create file if not existing.
// write data
// close file
// release lock
func newDocument(dockey string, fpath, fname string, data map[string][]byte) (doc *document, err error) {
	defer panics.Recover(&err)

	filename, err := assertSystemObjectPath(fpath, fname) // panics

	// acquire lock for file
	lock, ok, e := LockResource(filename, "create document "+string(dockey))
	panics.OnError(e, "lockResource:", dockey, filename)
	panics.OnFalse(ok, "lockResource:", dockey, filename)
	defer lock.Unlock()

	file, e := createSystemFile(filename)
	panics.OnError(e, "OpenFile:", filename)
	defer file.Close()

	info, e := file.Stat()
	panics.OnError(e, "Unexpected fault:", "Stat", filename)

	// record specific
	records := make(map[string][]byte, len(data))
	doc = &document{dockey, &info, time.Now(), records, lock, false}
	for k, v := range data {
		records[k] = v
	}
	e = doc.Write(file)
	panics.OnError(e, "doc.Write:")

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

func (d *document) String() string {
	return string(d.Bytes())
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

// Saves document, if dirty.
// Write Lock acquired for duration (attempted)
// New document file is atomically swapped.
func updateDocument(doc *document, filename string) (ok bool, err error) {
	defer panics.Recover(&err)

	// nop if doc hasn't changed.
	if !doc.IsDirty() {
		return true, nil
	}

	// create temp file
	swapfile := filename + ".new"
	file, e := os.OpenFile(swapfile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(0644))
	panics.OnError(e, "os.OpenFile:", swapfile)
	defer file.Close()

	e = doc.Write(file)
	panics.OnError(e, "updateDocument:", "doc.Write:")

	// acquire lock for doc file
	lock, ok, e := LockResource(filename, "create document "+string(doc.key))
	panics.OnError(e, "lockResource:", doc.key, filename)
	panics.OnFalse(ok, "lockResource:", doc.key, filename)
	defer lock.Unlock()

	e = os.Remove(filename)
	panics.OnError(e, "os.Remove:", filename)

	e = os.Rename(swapfile, filename)
	panics.OnError(e, "os.Rename:", swapfile, filename)

	return true, nil
}

// load for read.
// read file and closes it.
// REVU TODO insure that this method can only be invoked inside an exlusive section
func loadDocument(dockey string, filename string) (doc *document, err error) {
	defer panics.Recover(&err)

	// verify document file
	info, e := os.Stat(filename)
	panics.OnError(e, "os.Stat", filename)
	panics.OnTrue(info.IsDir(), filename, "is file")

	// REVU: lock checks could go here.

	// open and defer close document file
	file, e := os.Open(filename)
	panics.OnError(e, "os.OpenFile", filename)
	defer file.Close()

	// read document file
	bufsize := int(info.Size())
	buf := make([]byte, bufsize)
	n, e := file.Read(buf)
	panics.OnError(e, "file.Read")
	panics.OnTrue(n < bufsize, "file.Read", "partial read")

	// create and load document
	doc = &document{dockey, &info, time.Now(), make(map[string][]byte), nil, false}
	lines := strings.Split(string(buf), "\n")
	for _, line := range lines {
		if len(line) > 0 && line[0] != '#' {
			//			log.Printf("%s\n", line)
			tuple2 := strings.SplitN(line, ":", 2)
			panics.OnFalse(len(tuple2) == 2, "malformed record", line)

			// trim all whitespace from key and value
			tuple2[0] = strings.Trim(tuple2[0], "\t ")
			tuple2[1] = strings.Trim(tuple2[1], "\t ")
			doc.records[tuple2[0]] = []byte(tuple2[1])
		}
	}

	return
}

func deleteDocument(dockey string, filename string) (ok bool, err error) {
	defer panics.Recover(&err)

	// verify document file
	info, e := os.Stat(filename)
	panics.OnError(e, "os.Stat", filename)
	panics.OnTrue(info.IsDir(), filename, "must be file")

	// acquire lock for file
	lock, ok, e := LockResource(filename, string(dockey))
	panics.OnError(e, "lockResource:", dockey, filename)
	panics.OnFalse(ok, "lockResource:", dockey, filename)
	defer lock.Unlock()

	e = os.Remove(filename)
	panics.OnError(e, "os.Remove", filename)

	return true, nil
}
