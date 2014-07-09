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

package command

import (
	"github.com/elasticsearch/kriterium/panics"
	"log"
	"lsf"
	"lsf/schema"
)

const listStreamCmdCode lsf.CommandCode = "stream-list"

type listStreamOptionsSpec struct {
	global  BoolOptionSpec
	verbose BoolOptionSpec
}

var listStream *lsf.Command
var listStreamOptions *listStreamOptionsSpec

func init() {

	listStream = &lsf.Command{
		Name:  listStreamCmdCode,
		About: "List Streams defined ",
		Run:   runListStream,
		Flag:  FlagSet(listStreamCmdCode),
	}
	listStreamOptions = &listStreamOptionsSpec{
		global:  NewBoolFlag(listStream.Flag, "G", "global", false, "global scope flag for command", false),
		verbose: NewBoolFlag(listStream.Flag, "v", "verbose", false, "detailed output", false),
	}
}

func runListStream(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)

	verbose := *listStreamOptions.verbose.value
	v, found := env.Get(streamOptionVerbose)
	if found {
		verbose = verbose || v.(bool)
	}

	digests := env.GetResourceDigests("stream", verbose, schema.LogStreamDigest)
	for _, digest := range digests {
		log.Println(digest)
	}

	return nil
}
