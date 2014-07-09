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
	"github.com/elasticsearch/kriterium/panics"
	"os"
	"path"
	"path/filepath"
)

// Returns []string{} if pattern is ""/zv.
// REVU:
// if sampling period is (e.g. 1/microsec) extremely small
// then os.Stat may hit a phantom file: it was deleted right
// after filepath.Glob returned. There is nothing that can
// be done here but to ignore it.
func FindMatchingPaths(basepath, pattern string) (matches []string, err error) {
	defer panics.Recover(&err)

	if pattern == "" {
		return []string{}, nil
	}

	glob := path.Join(basepath, pattern)

	fspaths0, e := filepath.Glob(glob)
	panics.OnError(e, "FindMatchingPaths", "path.Join", glob)

	fspaths := make([]string, len(fspaths0))
	n := 0
	for _, fspath := range fspaths0 {
		_, e := os.Stat(fspath)
		switch {
		case e == nil:
			fspaths[n] = fspath
			n++
		case os.IsNotExist(e): // ignore - see REVU note
		default: // what is it?
			panics.OnError(e, "FindMatchingPaths", "os.Stat", fspath)
		}
	}
	return fspaths[:n], nil
}
