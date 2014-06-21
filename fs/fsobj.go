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
	// REVU: todo rename this or just do ModTime() time.Time
	// returns 'age' since last mod time.
	Age() time.Duration
	// returns time Object info was recorded
	Timestamp() time.Time
	// returns 'info age' since info was recorded
	InfoAge() time.Duration
	//
	Flags() uint8
	SetFlags(flags uint8)
}

func SameObject(a, b Object) bool {
	return os.SameFile(a.Info(), b.Info())
}

func Renamed(a, b Object) (bool, error) {
	if !SameObject(a, b) {
		return false, errors.New("not same object")
	}
	return a.Info().Name() != b.Info().Name(), nil
}

// panics
func Renamed0(a, b Object) bool {
	res, e := Renamed(a, b)
	if e != nil {
		panic(errors.New("not same object"))
	}
	return res
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
	flags    uint8       // 8bit user flag field
}

// AsObject constructs an object instance for the given info.
// Returns nil for nil.
// object InfoTime is set to AsObject function call time.
func AsObject(info os.FileInfo) Object {
	if info == nil {
		return nil
	}
	return &object{info, oid(info), time.Now(), 0}
}

// AsObject constructs an object instance for the given info.
// Returns nil for nil.
func AsObjectAt(info os.FileInfo, infotime time.Time) Object {
	if info == nil {
		return nil
	}
	return &object{info, oid(info), infotime, 0}
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

func (obj *object) SetFlags(flags uint8) {
	obj.flags = flags
}

func (obj *object) Flags() uint8 {
	return obj.flags
}

// Pretty Print
func (obj *object) String() string {
	return fmt.Sprintf(
		"fsobject %s:id %12d:info-age (nsec) %12d:size (b) %12d:age (nsec) %20q:name flags:%b",
		obj.Id(), obj.InfoAge(), obj.Info().Size(), obj.Age(), obj.Info().Name(), obj.Flags())
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
