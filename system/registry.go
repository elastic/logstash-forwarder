package system

import (
	"lsf/panics"
	"os"
	"path"
)

// TODO: REVU if distinction between Registrar & Registry still makes sense.
// TODO: REVU if inclusion of System log api makes sense. (Why not in system?)
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
func openRegistry(dir string) (reg *registry, err error) {

	defer panics.Recover(&err)

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
	panics.OnError(e, "system.openRegistry:")

	info, e := root.Stat()
	panics.OnError(e, "system.openRegistry:")
	panics.OnFalse(info.IsDir(), "system.openRegistry:", dir, "must be directory")

	r := &registry{
		path:     rootpath,
		rootinfo: info,
	}

	return r, nil
}

// ----------------------------------------------------------------------------
// System Document Registry
// ----------------------------------------------------------------------------

// TODO: rename DocPathForKey
func DocpathForKey(lsfpath string, key DocId) (filepath, filename string) {
	id := string(key)
	return ObjectPathForId(lsfpath, id)
}

// REVU: these basically set the working path (only, as of now).
// TODO REVU if returning interface (and not support type) makes sense.
// e.g. *document exposes lock member but why not Document.Lock() .. ?

func (r *registry) updateDocument(doc *document) (bool, error) {
	docpath, docname := DocpathForKey(r.path, doc.key)
	return updateDocument(doc, path.Join(docpath, docname))
}

func (r *registry) readDocument(key DocId) (*document, error) {
	docpath, docname := DocpathForKey(r.path, key)
	return loadDocument(key, path.Join(docpath, docname))
}

func (r *registry) createDocument(key DocId, data map[string][]byte) (*document, error) {
	docpath, docname := DocpathForKey(r.path, key)
	return newDocument(key, docpath, docname, data)
}

func (r *registry) deleteDocument(key DocId) (bool, error) {
	docpath, docname := DocpathForKey(r.path, key)
	return deleteDocument(key, path.Join(docpath, docname))
}

// ----------------------------------------------------------------------------
// System Log Registry
// ----------------------------------------------------------------------------

// REVU: this can either do the cast to string as of now or actually implement
//       the log naming schema.
func LogPathForKey(lsfpath string, key LogId) (filepath, filename string) {
	id := string(key)
	return ObjectPathForId(lsfpath, id)
}

// TODO REVU if createLog, accessLog (mode), are sufficient here
// TODO REVU if returning interface (and not support type) makes sense.
// see ~ note for System Documents (above)

func (r *registry) accessLog(id LogId, mode LogAccessMode) (*syslog, error) {
	panic("not implemented")
}

func (r *registry) createLog(id LogId) (*syslog, error) {
	panic("not implemented")
}

func (r *registry) deleteLog(id LogId) (bool, error) {
	panic("not implemented")
}
