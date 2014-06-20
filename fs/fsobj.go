package fs

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"
)

type Object interface {
	// Hex encoded rep of object oid
	Id() string
	// Last recorded FileInfo - will not re-stat
	Info() os.FileInfo
	// String rep of Object
	String() string
	// returns 'age' since last mod time.
	Age() time.Duration
	// returns time Object info was recorded
	Timestamp() time.Time
	// returns 'info age' since info was recorded
	InfoAge() time.Duration
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

type object struct {
	info     os.FileInfo // associated file info struct
	oid      fsoid       // generated oid based on info.
	infotime time.Time   // time info (stat) recorded
}

// AsObject constructs an object instance for the given info.
// Returns nil for nil.
// object InfoTime is set to AsObject function call time.
func AsObject(info os.FileInfo) Object {
	if info == nil {
		return nil
	}
	return &object{info, oid(info), time.Now()}
}

// AsObject constructs an object instance for the given info.
// Returns nil for nil.
func AsObjectAt(info os.FileInfo, infotime time.Time) Object {
	if info == nil {
		return nil
	}
	return &object{info, oid(info), infotime}
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

func (obj *object) Age() time.Duration {
	return time.Now().Sub(obj.info.ModTime())
}

func (obj *object) Timestamp() time.Time {
	return obj.infotime
}

func (obj *object) InfoAge() time.Duration {
	return time.Now().Sub(obj.Timestamp())
}

// Pretty Print
func (obj *object) String() string {
	return fmt.Sprintf("fsobject id:%s info-age:%d (nsec) name:%s age:%d (nsec)", obj.Id(), obj.InfoAge(), obj.Info().Name(), obj.Age())
	//	return "fsobject id:" + obj.Id() + " name:" + obj.info.Name()
}

// (for now) using *nix oids as maximal. So,
// the max length for OID is dev (32bits) + ino (64b)
const OIDLength = 12

// REVU: this really needs to be a fixed sized array
type fsoid []byte

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
