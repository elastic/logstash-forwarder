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
	"flag"
	"lsf"
)

const remoteCmdCode lsf.CommandCode = "remote"

type remoteOptionsSpec struct {
	verbose BoolOptionSpec
	global  BoolOptionSpec
}

var Remote *lsf.Command
var remoteOptions *remoteOptionsSpec

const (
	remoteOptionVerbose = "command.remote.option.verbose"
	remoteOptionGlobal  = "command.remote.option.global"
)

func init() {

	Remote = &lsf.Command{
		Name:  remoteCmdCode,
		About: "Remote is a top level command for remote specific features of LSF",
		//		Init:  initialCmdEnv,
		Run:  runRemote,
		Flag: FlagSet(remoteCmdCode),
	}
	remoteOptions = &remoteOptionsSpec{
		verbose: NewBoolFlag(Remote.Flag, "v", "verbose", false, "be verbose in list", false),
		global:  NewBoolFlag(Remote.Flag, "G", "global", false, "command applies globally", false),
	}
}

func runRemote(env *lsf.Environment, args ...string) error {

	if *remoteOptions.verbose.value {
		env.Set(remoteOptionVerbose, true)
	}
	if *remoteOptions.global.value {
		env.Set(remoteOptionGlobal, true)
	}

	xoff := 0
	var subcmd *lsf.Command = listRemote
	if len(args) > 0 {
		xoff = 1
		switch lsf.CommandCode("remote-" + args[0]) {
		case addRemoteCmdCode:
			subcmd = addRemote
		case removeRemoteCmdCode:
			subcmd = removeRemote
		case updateRemoteCmdCode:
			subcmd = updateRemote
		case listRemoteCmdCOde:
			subcmd = listRemote
		default:
			// not panic -- return error TODO
			panic("BUG - unknown subcommand for remote: " + args[0])
		}
	}

	return lsf.Run(env, subcmd, args[xoff:]...)
}

func verifyMinimalRemoteRequiredOpts(env *lsf.Environment, args ...string) error {
	if e := verifyRequiredOption(removeRemoteOptions.id); e != nil {
		return e
	}
	return nil
}

type editRemoteOptionsSpec struct {
	verbose BoolOptionSpec
	global  BoolOptionSpec
	id      StringOptionSpec
	host    StringOptionSpec
	port    Int64OptionSpec
}

func initEditRemoteOptionsSpec(flagset *flag.FlagSet) *editRemoteOptionsSpec {
	return &editRemoteOptionsSpec{
		verbose: NewBoolFlag(flagset, "v", "verbose", false, "be verbose", false),
		global:  NewBoolFlag(flagset, "G", "global", false, "global scope flag for command", false),
		id:      NewStringFlag(flagset, "r", "remote-id", "", "unique identifier for remote port", true),
		host:    NewStringFlag(flagset, "h", "remote-host", "", "URL of the remote port", true),
		port:    NewInt64Flag(flagset, "p", "remote-port", 0, "IP port number of remote port", true),
	}
}
