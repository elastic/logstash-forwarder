package fs

import (
	"encoding/hex"
	"os"
)

type Object interface {
	Id() string        // hex rep of object.oid
	Info() os.FileInfo // associated fileInfo.
}

func SameObject(a, b Object) bool {
	return os.SameFile(a.Info(), b.Info())
}

// Return an os agnostic hex representation of
// the unique id of this FS Object
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
