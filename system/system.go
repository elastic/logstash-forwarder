package system

import (
	"errors"
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
