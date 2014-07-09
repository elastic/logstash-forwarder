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
	"lsf/test" // REVU: TODO use kriterium
	"path"
	"testing"
)

// TEST:
// nil/zero-value (zv) args should result in error
func TestObjectPathForKeyInvalidArgs(t *testing.T) {
	// REVU: can this be automated
	assert := test.GetAssertionFor(t, "TestObjectPathForKeyInvalidArgs")

	lsfpath, oid := "", ""
	basepath, basename, e := objectPathForId(lsfpath, oid)
	assert.NotNil("e", e)
	assert.Nil("basepath", basepath)
	assert.Nil("basename", basename)

}

// TEST:
// No errors.
// Match expected path to provided OIDs.
func TestObjectPathForKey(t *testing.T) {

	// REVU: can this be automated
	assert := test.GetAssertionFor(t, "TestLogPathForKey")

	lsfpath := "/Users/alphazero/.lsf"
	// just test the 2 possible patterns:
	// 1 - top level resources
	// 2 - dot notation res ids
	var ids = []string{
		"system",
		"remote.remote-abc.remote",
		"stream.my-server.stream",
		"foo.bar.paz",
	}

	var expected = []struct {
		basepath, basename string
	}{
		{path.Join(lsfpath, "/"), "SYSTEM"},
		{path.Join(lsfpath, "remote/remote-abc"), "REMOTE"},
		{path.Join(lsfpath, "stream/my-server"), "STREAM"},
		{path.Join(lsfpath, "foo/bar"), "PAZ"},
	}

	for n, id := range ids {
		basepath, basename, err := objectPathForId(lsfpath, id)
		assert.Nil("err", err)
		assert.StringsEqual("basepath", expected[n].basepath, basepath)
		assert.StringsEqual("basename", expected[n].basename, basename)
	}
}
