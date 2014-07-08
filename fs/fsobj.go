// Licensed to Elasticsearch under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package fs

import (
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"time"
)

// ----------------------------------------------------------------------
// OS specific
// ----------------------------------------------------------------------

// --- os specific ----------------------------------------- BEGIN
// TODO: move this section to os specific files if necessary.

// c.f. http://en.wikipedia.org/wiki/General_Parallel_File_System
const MAX_FNAME_LEN = 255

// (for now) using *nix oids as maximal. So,
// the max length for OID is dev (4b) + ino (8b)
const OID_LEN = 12

// --- os specific ----------------------------------------- END

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

// ----------------------------------------------------------------------
// Helper functions
// ----------------------------------------------------------------------

func SameObject(a, b Object) bool {
	return os.SameFile(a.Info(), b.Info())
}

func Renamed(a, b Object) (bool, error) {
	if !SameObject(a, b) {
		return false, ERR.NOT_SAME_OBJECT()
	}
	return a.Info().Name() != b.Info().Name(), nil
}

// panics
func Renamed0(a, b Object) bool {
	res, e := Renamed(a, b)
	if e != nil {
		panic(ERR.NOT_SAME_OBJECT())
	}
	return res
}

func Modified(a, b Object) (bool, error) {
	if !SameObject(a, b) {
		return false, ERR.NOT_SAME_OBJECT()
	}
	ainfo, binfo := a.Info(), b.Info()

	return ainfo.Size() != binfo.Size() || ainfo.ModTime() != binfo.ModTime(), nil
}

// panics
func Modified0(a, b Object) bool {
	res, e := Modified(a, b)
	if e != nil {
		panic(ERR.NOT_SAME_OBJECT())
	}
	return res
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

// ----------------------------------------------------------------------
// FileSystem Object
// ----------------------------------------------------------------------

// fs.Object defines a cross platform abstraction of FS objects, such as files.
// Main purpose is to paper over the differences in various OS/FS (systems).
type Object interface {
	// Hex encoded rep of object oid
	Id() string
	// Last recorded FileInfo - will not re-stat
	Info() os.FileInfo
	// String rep of Object
	String() string
	// String rep of Object with labels.
	Debug() string
	// returns 'age' since last mod time.
	Age() time.Duration
	// returns time Object info was recorded
	Timestamp() time.Time
	// returns 'info age' since info was recorded
	InfoAge() time.Duration
	// returns the associated user flags for this fsobject.
	// Note that flags are purely specific to the reference and not the underlying filesystem object.
	Flags() uint8
	// Sets the associated user flags for this fsobject.
	// Note that flags are purely specific to the reference and not the underlying filesystem object.
	SetFlags(flags uint8)
}

// ----------------------------------------------------------------------
// FileSystem Object: Ref. Impl.
// ----------------------------------------------------------------------

type object struct {
	info     os.FileInfo // associated file info struct
	oid      fsoid       // generated oid based on info.
	infotime time.Time   // time info (stat) recorded
	flags    uint8       // 8bit user flag field
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

var max_fname_len_str = fmt.Sprintf("%d", MAX_FNAME_LEN)
var max_id_len_str = fmt.Sprintf("%d", OID_LEN)
var debugFmt = "fsobject %" + max_id_len_str + "s:id flags:%b %12d:info-age (nsec) %12d:size (b) %12d:age (nsec) %-" + max_fname_len_str + "q:name"
var normalFmt = "fsobject %" + max_id_len_str + "s %b %12d %12d %12d %-" + max_fname_len_str + "q"

// pretty print with field labels.
func (obj *object) Debug() string {
	return fmt.Sprintf(
		debugFmt,
		obj.Id(), obj.Flags(), obj.InfoAge(), obj.Info().Size(), obj.Age(), obj.Info().Name())
}

func (obj *object) String() string {
	return fmt.Sprintf(
		normalFmt,
		obj.Id(), obj.Flags(), obj.InfoAge(), obj.Info().Size(), obj.Age(), obj.Info().Name())
}

// ----------------------------------------------------------------------
// FileSystem ObjectMap
// ----------------------------------------------------------------------

type objectIterationOrder string

var ObjectIterationOrder = struct {
	ById, ByAge objectIterationOrder
}{
	ById:  objectIterationOrder("order-by-id"),
	ByAge: objectIterationOrder("order-by-age"),
}

// REVU: this is fairly generic. Move to lsf/misc.go
type iterationDirection byte

var IterationDirection = struct {
	Ascending, Descending iterationDirection
}{
	Ascending:  iterationDirection(0),
	Descending: iterationDirection(1),
}

// ObjectMap provides methods for ordered iteration over
// maps of FS Objects.
type ObjectMap interface {
	// Returns the map's value set sorted per input args.
	// Equiv to iterating over the underlying map via ranging over ObjectMap.Keys()
	Sort(order objectIterationOrder, direction iterationDirection) []Object
	// The actual map.
	RawMap() map[string]Object
}

// ----------------------------------------------------------------------
// FileSystem ObjectMap: Ref. Impl.
// ----------------------------------------------------------------------

type ByAgeAscending []Object

func (a ByAgeAscending) Len() int      { return len(a) }
func (a ByAgeAscending) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByAgeAscending) Less(i, j int) bool {
	return a[i].Age() < a[j].Age()
}

type ByAgeDescending []Object

func (a ByAgeDescending) Len() int      { return len(a) }
func (a ByAgeDescending) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByAgeDescending) Less(i, j int) bool {
	return a[i].Age() > a[j].Age()
}

type ByIdAscending []Object

func (a ByIdAscending) Len() int      { return len(a) }
func (a ByIdAscending) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByIdAscending) Less(i, j int) bool {
	return a[i].Id() < a[j].Id()
}

type ByIdDescending []Object

func (a ByIdDescending) Len() int      { return len(a) }
func (a ByIdDescending) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByIdDescending) Less(i, j int) bool {
	return a[i].Id() > a[j].Id()
}

type objectMap map[string]Object

func AsObjectMap(m map[string]Object) ObjectMap {
	return objectMap(m)
}

func (m objectMap) RawMap() map[string]Object {
	return map[string]Object(m)
}

func (m objectMap) Sort(order objectIterationOrder, direction iterationDirection) []Object {
	var objects []Object
	for _, object := range m {
		objects = append(objects, object)
	}
	switch {
	case order == ObjectIterationOrder.ByAge && direction == IterationDirection.Ascending:
		sort.Sort(ByAgeAscending(objects))

	case order == ObjectIterationOrder.ByAge && direction == IterationDirection.Descending:
		sort.Sort(ByAgeDescending(objects))

	case order == ObjectIterationOrder.ById && direction == IterationDirection.Ascending:
		sort.Sort(ByIdAscending(objects))

	case order == ObjectIterationOrder.ById && direction == IterationDirection.Descending:
		sort.Sort(ByIdDescending(objects))

	default:
		panic("BUG - unknown sort order")
	}
	return objects
}
