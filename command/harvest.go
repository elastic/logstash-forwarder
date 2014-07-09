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
	"lsf"
)

const cmd_harvest lsf.CommandCode = "harvest"

type harvestOptionsSpec struct {
	path   StringOptionSpec
	stream StringOptionSpec
}

var Harvest *lsf.Command
var harvestOptions *harvestOptionsSpec

func init() {

	Harvest = &lsf.Command{
		Name: cmd_harvest,
		Run:  runHarvest,
		Flag: FlagSet(cmd_harvest),
	}

	harvestOptions = &harvestOptionsSpec{
		path:   NewStringOptionSpec("p", "path", ".", "path to log-stream files", false),
		stream: NewStringOptionSpec("s", "stream", "", "the log-stream identifier", false),
	}
	harvestOptions.path.defineFlag(Harvest.Flag)
	harvestOptions.stream.defineFlag(Harvest.Flag)
}

func runHarvest(env *lsf.Environment, args ...string) error {
	/*
		prospecter.GoHarvest(in, out, err, stream, path,
	*/
	//	env.Vars["some.Key()"]

	panic("command.harvest() not impelemented!")

}
