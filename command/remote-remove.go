package command

import (
	"log"
	"lsf"
)

const cmd_remote_remove lsf.CommandCode = "remote-remove"

type removeRemoteOptionsSpec struct {
	global BoolOptionSpec
}

var removeRemote *lsf.Command
var removeRemoteOptions *removeRemoteOptionsSpec

func init() {

	removeRemote = &lsf.Command{
		Name:  cmd_remote_remove,
		About: "Remove a new log remote",
		Run:   runRemoveRemote,
		Flag:  FlagSet(cmd_remote_remove),
	}
	removeRemoteOptions = &removeRemoteOptionsSpec{
		global: NewBoolFlag(removeRemote.Flag, "g", "gg", false, "ggg", false),
	}
}

func runRemoveRemote(env *lsf.Environment, args ...string) error {
	for _, arg := range args {
		log.Printf("arg: %s\n", arg)
	}
	panic("command.runRemoveRemote() not impelemented!")
}
