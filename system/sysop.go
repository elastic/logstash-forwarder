package system

import (
	"fmt"
	"path"
	"strings"
)

// sysop contains the impl. for the protocol (e.g. locking)
// required for various system operations.

// explicit type the OpCode
// and keep it package visibility only
type OpCode string

var Op = struct {
	LsfNew,

	StreamAdd,
	StreamUpdate,
	StreamRemove,
	StreamList,
	StreamTrack,

	RemoteAdd,
	RemoteUpdate,
	RemoteRemove,
	RemoteList,

	_stub OpCode
}{
	StreamAdd: OpCode("lsf-new"),

	StreamAdd:    OpCode("stream-add"),
	StreamUpdate: OpCode("stream-update"),
	StreamRemove: OpCode("stream-remove"),
	StreamList:   OpCode("stream-list"),
	StreamTrack:  OpCode("stream-track"),

	RemoteAdd:    OpCode("remote-add"),
	RemoteUpdate: OpCode("remote-update"),
	RemoteRemove: OpCode("remote-remove"),
	RemoteList:   OpCode("remote-list"),
}

// Returns an operation lock for the specified operation on/for the
// given resource (resId) of the specified LSF environment (portPath).
// meta is optional
// Treats errors as fatal and panics.
// If exclusive op is already in progress, will return an error.
// This is the only error that it will return.

func ExclusiveResourceOp(portPath string, opcode OpCode, resId string, meta string) (opLock Lock, lockId string, err error) {
	opelems := strings.Split(string(opcode), "-")
	subject := opelems[0]
	action := opelems[1]

	// this
	s := []byte{}
	s = append([]byte(s), subject...)
	s = append([]byte(s), "."...)
	s = append([]byte(s), resId...)
	s = append([]byte(s), "."...)
	s = append([]byte(s), action...)

	resourceOp := string(s)

	lockId = path.Join(portPath, resourceOp)
	opLock, ok, e := LockResource(lockId, meta)
	if e != nil {
		panic(e)
	}
	if !ok {
		err = fmt.Errorf("%s already in progress", resourceOp)
		return
	}

	return
}
