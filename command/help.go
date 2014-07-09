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

const cmd_help lsf.CommandCode = "help"

var Help *lsf.Command
var helpOptions *helpOptionsSpec

type helpOptionsSpec struct {
	command StringOptionSpec
}

func init() {

	Help = &lsf.Command{
		Name:  cmd_help,
		About: "Provides usage information for LS/F commands",
		Run:   runHelp,
		Flag:  FlagSet(cmd_help),
		Usage: "help <command>",
	}

	helpOptions = &helpOptionsSpec{
		command: NewStringOptionSpec("c", "command", "", "the command you need help with", true),
	}
	helpOptions.command.defineFlag(Help.Flag)
}

func runHelp(env *lsf.Environment, args ...string) error {
	panic("command.help() not impelemented!")
}
