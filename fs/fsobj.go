package fs

import (
	"encoding/hex"
	"errors"
	"os"
)

type Object interface {
	Id() string        // hex rep of object.oid
	Info() os.FileInfo // associated fileInfo.
	String() string
}

func SameObject(a, b Object) bool {
	return os.SameFile(a.Info(), b.Info())
}

func Modified(a, b Object) (bool, error) {
	if !SameObject(a, b) {
		return false, errors.New("not same object")
	}
	ainfo, binfo := a.Info(), b.Info()

	return ainfo.Size() != binfo.Size() || ainfo.ModTime() != binfo.ModTime(), nil
}

// panics
func Modified0(a, b Object) bool {
	res, e := Modified(a, b)
	if e != nil {
		panic(errors.New("not same object"))
	}
	return res
}

// Return an os agnostic hex representation of
// the unique id of this FS Object.
// REVU TODO fix the length
func (obj *object) Id() string {
	return hex.EncodeToString(obj.oid)
}

// Return the associated os.FileInfo
func (obj *object) Info() os.FileInfo {
	return obj.info
}

// Pretty Print
func (obj *object) String() string {
	return "fsobject id:" + obj.Id() + " name:" + obj.info.Name()
}

// Returns nil for nil.
func AsObject(info os.FileInfo) Object {
	if info == nil {
		return nil
	}
	return &object{info, oid(info)}
}

// (for now) using *nix oids as maximal. So,
// the max length for OID is dev (32bits) + ino (64b)
const OIDLength = 12

// REVU: this really needs to be a fixed sized array
type fsoid []byte

// for convenience/efficiency. so we compute the oid once.
type object struct {
	info os.FileInfo // associated file info struct
	oid  fsoid       // generated oid based on info.
}

// Returns true if the (base named) file is an ignorable FS artifact.
// (For example, on *nix, fs.Ingore(".") returns true)
func Ignore(fname string) bool {
	for _, f := range StdIgnores() {
		if fname == f {
			return true
		}
	}
	return false
}

// Returns the list of standard ignore list for the FS.
// See Ignore()
func StdIgnores() []string {
	return ignoredFiles()
}
