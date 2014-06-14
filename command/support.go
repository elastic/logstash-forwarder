package command

import (
	"lsf"
	"lsf/system"
	"lsf/anomaly"
)

// support functions for commands
// NOTE: all functions with names beginning with _ throw panics

// lock resource
func _lockResource(env *lsf.Environment, resource string, reason string) system.Lock {
	lockid := env.ResourceId(resource)
	lock, ok, e := system.LockResource(lockid, reason)
	anomaly.PanicOnError(e, "command.runAddStream:", "lockResource:")
	anomaly.PanicOnFalse(ok, "command.runAddStream:", "lockResource:", lockid)

	return lock
}

// assert resource does not exist
func _assertNotExists (env *lsf.Environment, docid system.DocId) {
	doc, e := env.LoadDocument(docid)
	if e == nil && doc != nil {
		panic(lsf.E_EXISTING)
	}
}

// assert resource does not exist
func _assertExists (env *lsf.Environment, docid system.DocId) {
	doc, e := env.LoadDocument(docid)
	if e != nil || doc == nil {
		panic(lsf.E_NOTEXISTING)
	}
}
