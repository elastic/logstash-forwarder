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

package system

import (
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
	LsfNew: OpCode("lsf-new"),

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
		err = ERR.SYSTEM_OP_FAILURE("ExclusiveResourceOp:", resourceOp, "already in progress")
		return
	}
	return
}
