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
	"fmt"
	"os"
	"time"
)

// structure of the FS based lock
type lock struct {
	resource string
	accessed time.Time // recorded - not used
	deadline time.Time // recorded - not used
	data     []byte
	fileinfo *os.FileInfo
}

type Lock interface {
	Unlock() error
}

func (l *lock) Unlock() error {
	if l == nil {
		panic("BUG lock.Unlock: nil receiver")
	}
	if l.fileinfo == nil {
		return ERR.INVALID_LOCK_STATE("resource:", l.resource, "fileinfo is nil")
	}
	if l.resource == "" {
		panic(ERR.INVALID_LOCK_STATE("BUG lock.Unlock: nil resource", "resource:", l.resource))
	}

	e := removeLockFile(l.resource)
	if e != nil {
		return ERR.SYSTEM_OP_FAILURE("removeLockFile:")
	}

	l.fileinfo = nil
	l.data = nil
	l.accessed = time.Now()

	return nil
}

// attempts to create a lock file for the named resource.
// returns:
// lock, true, nil on success
// nil, false, nil on resource already locked
// nil, false, error on error
func LockResource(resource string, info string) (l Lock, ok bool, err error) {

	//	log.Printf("DEBUG LockResource: %s", resource)
	// try creating lock file
	// REVU: this can be reenterant TODO check the pid in lock file.
	file, e := createLockFile(resource)
	if os.IsExist(e) {
		return nil, false, nil
	} else if e != nil {
		return nil, false, e
	}
	//	log.Printf("DEBUG LockResource: %", file.Name())

	fileinfo, e := file.Stat()
	if e != nil {
		return nil, false, e // BUG
	}

	now := time.Now()
	data := []byte(fmt.Sprintf("%d %d %d %q\n", now.UnixNano(), now.UnixNano(), os.Getpid(), info))

	n, e := file.Write(data)
	//	file.Sync()
	if e != nil {
		return nil, false, e
	}
	if e == nil && n < len(data) {
		err = ERR.SYSTEM_OP_FAILURE("LockResource:", "file.Write", "n:", n, "data-len:", len(data))
		return nil, false, err
	}
	if e != nil {
		err = ERR.SYSTEM_OP_FAILURE("LockResource:", "file.Write", "resource:", resource, "cause:", e.Error())
		return nil, false, err
	}

	defer file.Close()

	return &lock{resource, now, now, data, &fileinfo}, true, nil
}

func removeLockFile(resource string) error {
	//	log.Printf("DEBUG removeLockFile: %s", resource)
	filename := resource + ".lock"
	//	log.Printf("DEBUG removeLockFile: %s", filename)
	if _, e := os.Stat(filename); os.IsNotExist(e) {
		return e
	}
	if e := os.Chmod(filename, os.FileMode(0644)); e != nil {
		return e
	}
	if e := os.Remove(filename); e != nil {
		return e
	}
	return nil
}

func createLockFile(resource string) (*os.File, error) {
	//	log.Printf("DEBUG createLockFile: %s", resource)
	filename := resource + ".lock"
	//	log.Printf("DEBUG createLockFile: %s", filename)
	file, e := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(0644))
	return file, e
}
