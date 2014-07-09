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

const listRemoteCmdCOde lsf.CommandCode = "remote-list"

type listRemoteOptionsSpec struct {
	global  BoolOptionSpec
	verbose BoolOptionSpec
}

var listRemote *lsf.Command
var listRemoteOptions *listRemoteOptionsSpec

func init() {

	listRemote = &lsf.Command{
		Name:  listRemoteCmdCOde,
		About: "List Remotes defined ",
		Run:   runListRemote,
		Flag:  FlagSet(listRemoteCmdCOde),
	}
	listRemoteOptions = &listRemoteOptionsSpec{
		global:  NewBoolFlag(listRemote.Flag, "G", "global", false, "global scope flag for command", false),
		verbose: NewBoolFlag(listRemote.Flag, "v", "verbose", false, "detailed output", false),
	}
}

func runListRemote(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)

	verbose := *listRemoteOptions.verbose.value
	v, found := env.Get(remoteOptionVerbose)
	if found {
		verbose = verbose || v.(bool)
	}

	digests := env.GetResourceDigests("remote", verbose, schema.PortDigest)
	for _, digest := range digests {
		log.Println(digest)
	}

	return nil
}
