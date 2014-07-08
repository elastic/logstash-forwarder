package command

import (
	"github.com/elasticsearch/kriterium/panics"
	"lsf"
)

const removeRemoteCmdCode lsf.CommandCode = "remote-remove"

type removeRemoteOptionsSpec struct {
	global BoolOptionSpec
	id     StringOptionSpec
}

var removeRemote *lsf.Command
var removeRemoteOptions *removeRemoteOptionsSpec

func init() {

	removeRemote = &lsf.Command{
		Name:  removeRemoteCmdCode,
		About: "Remove a new log remote",
		Init:  verifyMinimalRemoteRequiredOpts,
		Run:   runRemoveRemote,
		Flag:  FlagSet(removeRemoteCmdCode),
	}
	removeRemoteOptions = &removeRemoteOptionsSpec{
		global: NewBoolFlag(removeRemote.Flag, "G", "global", false, "global scope operation", false),
		id:     NewStringFlag(removeRemote.Flag, "r", "remote-id", "", "unique identifier for remote port", true),
	}
}

func runRemoveRemote(env *lsf.Environment, args ...string) (err error) {
	panics.Recover(&err)

	id := *removeRemoteOptions.id.value
	return env.RemoveRemotePort(id)
}
