package fs

import (
	"fmt"
	"log"
	"lsf/panics"
	"strconv"
	"time"
)

type InfoAge time.Duration

func (t *InfoAge) String() string {
	return fmt.Sprintf("%d", *t)
}
func (t *InfoAge) Set(vrep string) error {
	v, e := strconv.ParseInt(vrep, 10, 64)
	if e != nil {
		return e
	}
	var tt time.Duration = time.Duration(v) * time.Millisecond
	*t = InfoAge(tt)
	return nil
}

// fs.Object cache
type ObjectCache struct {
	options struct {
		maxSize uint16
		maxAge  InfoAge
	}
	Cache  map[string]Object
	gc     GcFunc
	gcArgs []interface{}
}

var GCAlgorithm = struct{ byAge, bySize GcFunc }{
	byAge:  AgeBasedGcFunc,
	bySize: SizeBasedGcFunc,
}

func NewFixedSizeObjectCache(maxSize uint16) *ObjectCache {
	oc := newObjectCache()
	oc.options.maxSize = maxSize
	oc.gc = SizeBasedGcFunc
	oc.gcArgs = []interface{}{oc.options.maxSize}
	return oc
}
func NewTimeWindowObjectCache(maxAge InfoAge) *ObjectCache {
	oc := newObjectCache()
	oc.options.maxAge = maxAge
	oc.gc = AgeBasedGcFunc
	oc.gcArgs = []interface{}{oc.options.maxAge}
	return oc
}
func newObjectCache() *ObjectCache {
	oc := new(ObjectCache)
	oc.Cache = make(map[string]Object)
	return oc
}
func (oc *ObjectCache) MarkDeleted(id string) bool {
	obj, found := oc.Cache[id]
	if !found {
		return false
	}
	obj.SetFlags(1)
	return true
}
func (oc *ObjectCache) IsDeleted(id string) (bool, error) {
	obj, found := oc.Cache[id]
	if !found {
		return false, fmt.Errorf("no such object")
	}

	return IsDeleted0(obj.Flags()), nil
}

func IsDeleted0(flag uint8) bool {
	return flag == uint8(1)
}

func (oc *ObjectCache) Gc() {
	n := oc.gc(oc.Cache, oc.gcArgs...)
	if n > 0 {
		log.Printf("GC: %d items removed - object-cnt: %d", n, len(oc.Cache))
	}
}

// REVU: TODO: sort these by age descending first
func AgeBasedGcFunc(cache map[string]Object, args ...interface{}) int {
	panics.OnFalse(len(args) == 1, "BUG", "AgeBasedGcFunc", "args:", args)
	limit, ok := args[0].(InfoAge)
	panics.OnFalse(ok, "BUG", "AgeBasedGcFunc", "limit:", args[0])
	n := 0
	for id, obj := range cache {
		if !IsDeleted0(obj.Flags()) {
			continue
		}
		if obj.Age() > time.Duration(limit) {
			delete(cache, id)
			n++
		}
	}
	return n
}

func SizeBasedGcFunc(cache map[string]Object, args ...interface{}) int {
	panics.OnFalse(len(args) == 1, "BUG", "SizeBasedGcFunc", "args:", args)
	limit, ok := args[0].(uint16)
	panics.OnFalse(ok, "BUG", "SizeBasedGcFunc", "limit:", args[0])
	n := 0
	if len(cache) <= int(limit) {
		return 0
	}
	for id, obj := range cache {
		if !IsDeleted0(obj.Flags()) {
			continue
		}
		delete(cache, id)
		n++
	}
	return n
}

type GcFunc func(cache map[string]Object, args ...interface{}) int
