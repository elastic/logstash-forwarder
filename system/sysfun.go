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
	"github.com/elasticsearch/kriterium/errors"
	"os"
	"path"
	"strings"
)

var NilValue = []byte{}

// TODO: must be OS portable
func UserHome() string {
	return os.Getenv("HOME")
}

// Return the canonical file basepath and basename for the identified
// system (FS) object. Nil/ZV input arg(s) results in error.
//
// Note: Don't confuse oid with fs.Object.Id(). oid is generalizing DocIds/LogIds
// which is merely a semantic identifier exclusive to the system.
func objectPathForId(lsfpath string, oid string) (basepath, basename string, err error) {
	if lsfpath == "" {
		err = errors.IllegalArgument("lsfpath:", "can not be zerovalue")
		return
	}
	if oid == "" {
		err = errors.IllegalArgument("oid:", "can not be zerovalue")
		return
	}

	keyparts := strings.Split(oid, ".")
	kplen := len(keyparts)
	switch kplen {
	case 1:
		return path.Join(lsfpath, basepath), strings.ToUpper(oid), nil
	default:
		docname := keyparts[kplen-1]
		basename = strings.Replace(oid, ".", "/", -1)[:len(oid)-len(docname)]
		return path.Join(lsfpath, basepath, basename), strings.ToUpper(docname), nil
	}
}

// REVU: the func name is wrong - this will do for now but needs to be addressed.
func assertSystemObjectPath(fpath, fname string) (filename string, err error) {
	dstat, e := os.Stat(fpath)
	if e != nil {
		// REVU: ok to create the directory
		e := os.MkdirAll(fpath, os.ModePerm)
		if e != nil {
			err = ERR.SYSTEM_OP_FAILURE("create sysobj path", fpath, "cause:", e.Error())
			return
		}
	} else if !dstat.IsDir() {
		err = errors.IllegalState("BUG", "not a directory:", fpath)
		return
	}
	filename = path.Join(fpath, fname)
	return filename, nil
}

func createSystemFile(filename string) (file *os.File, err error) {
	// REVU: hardcoded file mode ..
	return os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(0644))
}
