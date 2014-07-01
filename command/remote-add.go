package command

import (
	"lsf"
	"lsf/panics"
)

const addRemoteCmdCode lsf.CommandCode = "remote-add"

type addRemoteOptionsSpec struct {
	global BoolOptionSpec
	id     StringOptionSpec
	host   StringOptionSpec
	port   Int64OptionSpec
}

var addRemote *lsf.Command
var addRemoteOptions *editRemoteOptionsSpec

func init() {

	addRemote = &lsf.Command{
		Name:  addRemoteCmdCode,
		About: "Add a new remote port",
		Init:  _verifyAddRemoteRequiredOpts,
		Run:   runAddRemote,
		Flag:  FlagSet(addRemoteCmdCode),
	}
	addRemoteOptions = initEditRemoteOptionsSpec(addRemote.Flag)
}

func _verifyAddRemoteRequiredOpts(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)

	var e error
	e = verifyRequiredOption(addRemoteOptions.id)
	panics.OnError(e, "remote-add", "option", "id")

	e = verifyRequiredOption(addRemoteOptions.host)
	panics.OnError(e, "remote-add", "option", "host")

	e = verifyRequiredOption(addRemoteOptions.port)
	panics.OnError(e, "remote-add", "option", "port")

	return
}

func runAddRemote(env *lsf.Environment, args ...string) (err error) {

	id := *addRemoteOptions.id.value
	host := *addRemoteOptions.host.value
	port := int(*addRemoteOptions.port.value)

	return env.AddRemotePort(id, host, port)
}
