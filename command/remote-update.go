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
	"fmt"
	"github.com/elasticsearch/kriterium/panics"
	"lsf"
	"lsf/schema"
)

const updateRemoteCmdCode lsf.CommandCode = "remote-update"

var updateRemote *lsf.Command
var updateRemoteOptions *editRemoteOptionsSpec

func init() {

	updateRemote = &lsf.Command{
		Name:  updateRemoteCmdCode,
		About: "Update remote port sepc",
		Init:  verifyUpdateRemoteRequiredOpts,
		Run:   runUpdateRemote,
		Flag:  FlagSet(updateRemoteCmdCode),
	}
	updateRemoteOptions = initEditRemoteOptionsSpec(updateRemote.Flag)
}
func verifyUpdateRemoteRequiredOpts(env *lsf.Environment, args ...string) error {
	if e := verifyRequiredOption(updateRemoteOptions.id); e != nil {
		return e
	}
	return nil
}

func runUpdateRemote(env *lsf.Environment, args ...string) (err error) {
	panics.Recover(&err)

	id := *updateRemoteOptions.id.value
	updates := make(map[string][]byte)

	// update remote config document
	var option interface{}
	option = updateRemoteOptions.id
	if OptionProvided(option) {
		v := []byte(string(*updateRemoteOptions.id.value))
		updates[schema.PortElem.Id] = v
	}
	option = updateRemoteOptions.host
	if OptionProvided(option) {
		v := []byte(string(*updateRemoteOptions.host.value))
		updates[schema.PortElem.Host] = v
	}
	option = updateRemoteOptions.port
	if OptionProvided(option) {
		v := []byte(fmt.Sprintf("%d", (*updateRemoteOptions.port.value)))
		updates[schema.PortElem.PortNum] = v
	}

	return env.UpdateRemotePort(id, updates)
}
