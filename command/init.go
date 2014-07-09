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
)

const cmd_init lsf.CommandCode = "init"

type initOptionsSpec struct {
	home  StringOptionSpec
	force BoolOptionSpec
}

var Init *lsf.Command
var initOptions *initOptionsSpec

func init() {

	Init = &lsf.Command{
		Name:        cmd_init,
		About:       "Creates and initializes a LS/F port",
		Init:        nil_CommandInitFn,
		Run:         runInit,
		Flag:        FlagSet(cmd_init),
		Initializer: true,
	}
	initOptions = &initOptionsSpec{
		home:  NewStringFlag(Init.Flag, "h", "home", ".", "path directory to create lsf environemnt", false),
		force: NewBoolFlag(Init.Flag, "f", "force", false, "force the operation of command", false),
	}
}

// create and initialize an LSF base.
// The base will be created in the 'path' option location.
// 'force' flag must be set.
// Init in existing directory will raise error E_EXISTING
func runInit(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)

	home := lsf.AbsolutePath(*initOptions.home.value)
	force := *initOptions.force.value

	what := "Initialized"

	// init w/ existing is an error unless -force flag is set
	if env.Exists(home) {
		panics.OnFalse(force, "init.runInit:", "existing environment. use -force flag to reinitialize")
		what = "Re-Initialize"
	}

	envpath, e := lsf.CreateEnvironment(home, force)
	panics.OnError(e, "command/init.runInit", "on lsf.CreateEnvironment")

	log.Printf("%s LSF environment at %s\n", what, envpath)

	return
}
