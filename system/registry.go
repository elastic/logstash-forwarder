package system

import (
	"github.com/elasticsearch/kriterium/panics"
	"os"
	"path"
)

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

// REVU: these should formalize the XXX_LOG/XXX pattern for docs/logs.
// Panics on zero-value/nil 'key' arg.
func DocpathForKey(lsfpath string, key string) (filepath, filename string) {
	id := string(key)
	filepath, filename, e := objectPathForId(lsfpath, string(id))
	panics.OnError(e, "BUG")
	return
}

// REVU: these basically set the working path (only, as of now).
// TODO REVU if returning interface (and not support type) makes sense.
// e.g. *document exposes lock member but why not Document.Lock() .. ?

func (r *registry) updateDocument(doc *document) (bool, error) {
	docpath, docname := DocpathForKey(r.path, doc.key)
	return updateDocument(doc, path.Join(docpath, docname))
}

func (r *registry) readDocument(id string) (*document, error) {
	docpath, docname := DocpathForKey(r.path, id)
	return loadDocument(id, path.Join(docpath, docname))
}

func (r *registry) createDocument(id string, data map[string][]byte) (*document, error) {
	docpath, docname := DocpathForKey(r.path, id)
	return newDocument(id, docpath, docname, data)
}

func (r *registry) deleteDocument(id string) (bool, error) {
	docpath, docname := DocpathForKey(r.path, id)
	return deleteDocument(id, path.Join(docpath, docname))
}

// ----------------------------------------------------------------------------
// System Log Registry
// ----------------------------------------------------------------------------

// REVU: this can either do the cast to string as of now or actually implement
//       the log naming schema.
func LogPathForKey(lsfpath string, key string) (filepath, filename string) {
	id := string(key)

	filepath, filename, e := objectPathForId(lsfpath, id)
	panics.OnError(e, "BUG")
	return
}

// TODO REVU if createLog, accessLog (mode), are sufficient here
// TODO REVU if returning interface (and not support type) makes sense.
// see ~ note for System Documents (above)

func (r *registry) accessLog(id string, mode LogAccessMode) (*syslog, error) {
	panic("not implemented")
}

func (r *registry) createLog(id string) (*syslog, error) {
	panic("not implemented")
}

func (r *registry) deleteLog(id string) (bool, error) {
	panic("not implemented")
}
