package command

import (
	"log"
	"lsf"
)

const cmd_stream_remove lsf.CommandCode = "stream-remove"

type removeStreamOptionsSpec struct {
	global BoolOptionSpec
}

var removeStream *lsf.Command
var removeStreamOptions *removeStreamOptionsSpec

func init() {

	removeStream = &lsf.Command{
		Name:  cmd_stream_remove,
		About: "Remove a new log stream",
		Run:   runRemoveStream,
		Flag:  FlagSet(cmd_stream_remove),
	}
	removeStreamOptions = &removeStreamOptionsSpec{
		global: NewBoolFlag(removeStream.Flag, "g", "gg", false, "ggg", false),
	}
}

func runRemoveStream(env *lsf.Environment, args ...string) error {
	for _, arg := range args {
		log.Printf("arg: %s\n", arg)
	}
	panic("command.runRemoveStream() not impelemented!")
}
