package command

import (
	"log"
	"lsf"
)

const updateRemoteCmdCode lsf.CommandCode = "remote-update"

type updateRemoteOptionsSpec struct {
	global BoolOptionSpec
}

var updateRemote *lsf.Command
var updateRemoteOptions *updateRemoteOptionsSpec

func init() {

	updateRemote = &lsf.Command{
		Name:  updateRemoteCmdCode,
		About: "Update a new log remote",
		Run:   runUpdateRemote,
		Flag:  FlagSet(updateRemoteCmdCode),
	}
	updateRemoteOptions = &updateRemoteOptionsSpec{
		global: NewBoolFlag(updateRemote.Flag, "g", "global", false, "apply command in global context", false),
	}
}

func runUpdateRemote(env *lsf.Environment, args ...string) error {
	for _, arg := range args {
		log.Printf("arg: %s\n", arg)
	}
	panic("command.runUpdateRemote() not impelemented!")
}
