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
