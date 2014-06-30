package command

import (
	"lsf"
	//	"lsf/panics"
	//	"lsf/system"
)

// support functions for commands
// NOTE: all functions with names beginning with _ throw panics
// REVU: ALL can be env methods

// assert resource does not exist
func _assertNotExists(env *lsf.Environment, docId string) {
	doc, e := env.LoadDocument(docId)
	if e == nil && doc != nil {
		panic(lsf.E_EXISTING)
	}
}

// assert resource does not exist
func _assertExists(env *lsf.Environment, docId string) {
	doc, e := env.LoadDocument(docId)
	if e != nil || doc == nil {
		panic(lsf.E_NOTEXISTING)
	}
}
