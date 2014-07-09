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
	"log"
	"lsf"
)

const cmd_debug lsf.CommandCode = "debug"

var Debug *lsf.Command
var debugOptions *debugOptionsSpec

type debugOptionsSpec struct {
	command StringOptionSpec
}

func init() {

	Debug = &lsf.Command{
		Name:  cmd_debug,
		About: "Provides usage information for LS/F commands",
		Run:   runDebug,
		Flag:  FlagSet(cmd_debug),
		Usage: "debug <command>",
	}

	debugOptions = &debugOptionsSpec{
		command: NewStringOptionSpec("c", "command", "", "the command you need debug with", true),
	}
	debugOptions.command.defineFlag(Debug.Flag)
}

func runDebug(env *lsf.Environment, args ...string) error {

	debug("env.bound: %t", env.IsBound())
	debug0(env, "system.create-time")
	debug0(env, "stream.apache-123.scan.blocksize-byte")

	return nil
}

func debug0(env *lsf.Environment, record string) {
	value, e := env.GetRecord(record)
	if e != nil {
		debug("env.GetRecord(\"%s\") - NOT FOUND", record)
	} else {
		debug("env.GetRecord(\"%s\") => %q", record, string(value))
	}
}
func debug(fmt string, args ...interface{}) {
	log.Printf("command.Debug.debug: "+fmt, args...)
}
