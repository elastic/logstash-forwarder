package system

import (
	"fmt"
	"io"
	"log"
	"lsf/panics"
	"os"
	"path"
	"strings"
	"time"
)

var E_EXISTING_DOC = fmt.Errorf("document exists")

func DocpathForKey(lsfpath string, key DocId) (filepath, filename string) {
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

func (t DocId) String() string { return string(t) }

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

type DocumentDigestFn func(Document) string

// acquire resource lock
// create file if not existing.
// write data
// close file
// release lock
func newDocument(dockey DocId, fpath, fname string, data map[string][]byte) (doc *document, err error) {
	defer panics.Recover(&err)

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
	panics.OnError(e, "newDocument:", "lockResource:", dockey, filename)
	panics.OnFalse(ok, "newDocument:", "lockResource:", dockey, filename)
	defer lock.Unlock()

	_, e = os.Stat(filename)
	panics.OnFalse(os.IsNotExist(e), "newDocument:", filename)

	file, e := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(0644))
	panics.OnError(e, "newDocument:", "OpenFile:", filename)
	defer file.Close()

	//	log.Println("newDocument: created file %q", file)
	info, _ := file.Stat()

	records := make(map[string][]byte, len(data))
	doc = &document{dockey, &info, time.Now(), records, lock, false}
	for k, v := range data {
		records[k] = v
	}
	e = doc.Write(file)
	panics.OnError(e, "newDocument:", "doc.Write:")

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

// Saves document: if dirty, dirty flag cleared; otherwise returns false, nil.
// Write Lock acquired for duration (attempted)
// New document file is atomically swapped.
func updateDocument(doc *document, filename string) (ok bool, err error) {
	defer panics.Recover(&err)

	// create temp file
	swapfile := filename + ".new"
	file, e := os.OpenFile(swapfile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(0644))
	panics.OnError(e, "updateDocument:", "os.OpenFile:", swapfile)
	defer file.Close()

	e = doc.Write(file)
	panics.OnError(e, "updateDocument:", "doc.Write:")

	// acquire lock for doc file
	lock, ok, e := LockResource(filename, "create document "+string(doc.key))
	panics.OnError(e, "updateDocument:", "lockResource:", doc.key, filename)
	panics.OnFalse(ok, "updateDocument:", "lockResource:", doc.key, filename)
	defer lock.Unlock()

	e = os.Remove(filename)
	panics.OnError(e, "updateDocument:", "os.Remove:", filename)

	e = os.Rename(swapfile, filename)
	panics.OnError(e, "updateDocument:", "os.Rename:", swapfile, filename)

	log.Println("updateDocument: updated file %q", filename)

	return true, nil
}

// load for read.
// read file and closes it.
// REVU TODO what if locked?
func loadDocument(dockey DocId, filename string) (doc *document, err error) {
	defer panics.Recover(&err)

	// verify document file
	info, e := os.Stat(filename)
	panics.OnError(e, "loadDocument", "os.Stat", filename)
	panics.OnTrue(info.IsDir(), "loadDocument", filename, "is file")

	// REVU: lock checks could go here.

	// open and defer close document file
	file, e := os.Open(filename)
	panics.OnError(e, "loadDocument", "os.OpenFile", filename)
	defer file.Close()

	// read document file
	bufsize := int(info.Size())
	buf := make([]byte, bufsize)
	n, e := file.Read(buf)
	panics.OnError(e, "loadDocument", "file.Read")
	panics.OnTrue(n < bufsize, "loadDocument", "file.Read", "partial read")

	// create and load document
	doc = &document{dockey, &info, time.Now(), make(map[string][]byte), nil, false}
	lines := strings.Split(string(buf), "\n")
	for _, line := range lines {
		if len(line) > 0 && line[0] != '#' {
			//			log.Printf("%s\n", line)
			tuple2 := strings.SplitN(line, ":", 2)
			panics.OnFalse(len(tuple2) == 2, "loadDocument", "malformed record", line)

			// trim all whitespace from key and value
			tuple2[0] = strings.Trim(tuple2[0], "\t ")
			tuple2[1] = strings.Trim(tuple2[1], "\t ")
			doc.records[tuple2[0]] = []byte(tuple2[1])
		}
	}

	//	log.Println("newDocument: done")
	return
}

func deleteDocument(dockey DocId, filename string) (ok bool, err error) {
	defer panics.Recover(&err)

	// verify document file
	info, e := os.Stat(filename)
	panics.OnError(e, "system.deleteDocument:")
	panics.OnTrue(info.IsDir(), "system.deleteDocument:", filename, "must be file")

	// acquire lock for file
	lock, ok, e := LockResource(filename, "delete document "+string(dockey))
	panics.OnError(e, "deleteDocument:", "lockResource:", dockey, filename)
	panics.OnFalse(ok, "deleteDocument:", "lockResource:", dockey, filename)
	defer lock.Unlock()

	e = os.Remove(filename)
	panics.OnError(e, "system.deleteDocument:", "os.Remove", filename)

	return true, nil
}
