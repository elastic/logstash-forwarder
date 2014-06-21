package system

import (
	"lsf/panics"
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
