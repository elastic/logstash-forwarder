package system

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
)

var NilValue = []byte{}

// TODO: must be OS portable
func UserHome() string {
	return os.Getenv("HOME")
}

// Return the canonical file basepath and basename for the identified
// system (FS) object. Nil/ZV input arg(s) results in error.
//
// Note: Don't confuse oid with fs.Object.Id(). oid is generalizing DocId/LogId
// which is merely a semantic identifier exclusive to the system.
func objectPathForId(lsfpath string, oid string) (basepath, basename string, err error) {
	if lsfpath == "" || oid == "" {
		err = errors.New("zero-value arg")
		return
	}

	keyparts := strings.Split(oid, ".")
	kplen := len(keyparts)
	switch kplen {
	case 1:
		return path.Join(lsfpath, basepath), strings.ToUpper(oid), nil
	default:
		docname := keyparts[kplen-1]
		basename = strings.Replace(oid, ".", "/", -1)[:len(oid)-len(docname)]
		return path.Join(lsfpath, basepath, basename), strings.ToUpper(docname), nil
	}
}

// panics
func assertSystemObjectPath(fpath string) {
	dstat, e := os.Stat(fpath)
	if e != nil {
		// REVU: ok to create the directory
		e := os.MkdirAll(fpath, os.ModePerm)
		if e != nil {
			panic(fmt.Errorf("system: error creating dir %q - %s", fpath, e.Error()))
		}
	} else if !dstat.IsDir() {
		panic(fmt.Errorf("BUG - %s expected to be a directory", fpath))
	}
}
