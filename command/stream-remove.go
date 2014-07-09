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
	"lsf"
)

const removeStreamCmdCode lsf.CommandCode = "stream-remove"

type removeStreamOptionsSpec struct {
	global BoolOptionSpec
	id     StringOptionSpec
}

var removeStream *lsf.Command
var removeStreamOptions *removeStreamOptionsSpec

func init() {

	removeStream = &lsf.Command{
		Name:  removeStreamCmdCode,
		About: "Remove a new log stream",
		Init:  verifyRemoveStreamRequiredOpts,
		Run:   runRemoveStream,
		Flag:  FlagSet(removeStreamCmdCode),
	}
	removeStreamOptions = &removeStreamOptionsSpec{
		global: NewBoolFlag(removeStream.Flag, "G", "global", false, "global scope operation", false),
		id:     NewStringFlag(removeStream.Flag, "s", "stream-id", "", "unique identifier for stream", true),
	}
}

func verifyRemoveStreamRequiredOpts(env *lsf.Environment, args ...string) error {
	if e := verifyRequiredOption(removeStreamOptions.id); e != nil {
		return e
	}
	return nil
}

func runRemoveStream(env *lsf.Environment, args ...string) (err error) {
	panics.Recover(&err)

	id := *removeStreamOptions.id.value
	return env.RemoveLogStream(id)
}
