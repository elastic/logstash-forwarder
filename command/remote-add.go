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

const addRemoteCmdCode lsf.CommandCode = "remote-add"

type addRemoteOptionsSpec struct {
	global BoolOptionSpec
	id     StringOptionSpec
	host   StringOptionSpec
	port   Int64OptionSpec
}

var addRemote *lsf.Command
var addRemoteOptions *editRemoteOptionsSpec

func init() {

	addRemote = &lsf.Command{
		Name:  addRemoteCmdCode,
		About: "Add a new remote port",
		Init:  _verifyAddRemoteRequiredOpts,
		Run:   runAddRemote,
		Flag:  FlagSet(addRemoteCmdCode),
	}
	addRemoteOptions = initEditRemoteOptionsSpec(addRemote.Flag)
}

func _verifyAddRemoteRequiredOpts(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)

	var e error
	e = verifyRequiredOption(addRemoteOptions.id)
	panics.OnError(e, "remote-add", "option", "id")

	e = verifyRequiredOption(addRemoteOptions.host)
	panics.OnError(e, "remote-add", "option", "host")

	e = verifyRequiredOption(addRemoteOptions.port)
	panics.OnError(e, "remote-add", "option", "port")

	return
}

func runAddRemote(env *lsf.Environment, args ...string) (err error) {

	id := *addRemoteOptions.id.value
	host := *addRemoteOptions.host.value
	port := int(*addRemoteOptions.port.value)

	return env.AddRemotePort(id, host, port)
}
