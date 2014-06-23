package system

import (
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
// system (FS) object.
// Note: Don't confuse oid with fs.Object.Id(). oid is generalizing DocId/LogId
// which is merely a semantic identifier exclusive to the system.
func ObjectPathForId(lsfpath string, oid string) (basepath, basename string) {
	keyparts := strings.Split(oid, ".")
	kplen := len(keyparts)
	switch kplen {
	case 1:
		//		docname := keyparts[kplen-1]
		//		basename = strings.Replace(oid, ".", "/", -1)[:len(oid)-len(docname)]
		//		return path.Join(lsfpath, basepath, basename), strings.ToUpper(docname)
		return path.Join(lsfpath, basepath), strings.ToUpper(oid)
	default:
		docname := keyparts[kplen-1]
		basename = strings.Replace(oid, ".", "/", -1)[:len(oid)-len(docname)]
		return path.Join(lsfpath, basepath, basename), strings.ToUpper(docname)
	}
}
