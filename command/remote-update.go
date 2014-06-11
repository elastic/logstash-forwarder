package command

import (
	"log"
	"lsf"
)

const cmd_remote_update lsf.CommandCode = "remote-update"

type updateRemoteOptionsSpec struct {
	global BoolOptionSpec
}

var updateRemote *lsf.Command
var updateRemoteOptions *updateRemoteOptionsSpec

func init() {

	updateRemote = &lsf.Command{
		Name:  cmd_remote_update,
		About: "Update a new log remote",
		Run:   runUpdateRemote,
		Flag:  FlagSet(cmd_remote_update),
	}
	updateRemoteOptions = &updateRemoteOptionsSpec{
		global: NewBoolFlag(updateRemote.Flag, "g", "gg", false, "ggg", false),
	}
}

func runUpdateRemote(env *lsf.Environment, args ...string) error {
	for _, arg := range args {
		log.Printf("arg: %s\n", arg)
	}
	panic("command.runUpdateRemote() not impelemented!")
}
