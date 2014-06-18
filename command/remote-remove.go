package command

import (
	"log"
	"lsf"
)

const removeRemoteCmdCode lsf.CommandCode = "remote-remove"

type removeRemoteOptionsSpec struct {
	global BoolOptionSpec
}

var removeRemote *lsf.Command
var removeRemoteOptions *removeRemoteOptionsSpec

func init() {

	removeRemote = &lsf.Command{
		Name:  removeRemoteCmdCode,
		About: "Remove a new log remote",
		Run:   runRemoveRemote,
		Flag:  FlagSet(removeRemoteCmdCode),
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
