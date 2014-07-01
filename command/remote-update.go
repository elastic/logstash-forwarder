package command

import (
	"fmt"
	"lsf"
	"lsf/panics"
	"lsf/schema"
)

const updateRemoteCmdCode lsf.CommandCode = "remote-update"

var updateRemote *lsf.Command
var updateRemoteOptions *editRemoteOptionsSpec

func init() {

	updateRemote = &lsf.Command{
		Name:  updateRemoteCmdCode,
		About: "Update remote port sepc",
		Init:  verifyUpdateRemoteRequiredOpts,
		Run:   runUpdateRemote,
		Flag:  FlagSet(updateRemoteCmdCode),
	}
	updateRemoteOptions = initEditRemoteOptionsSpec(updateRemote.Flag)
}
func verifyUpdateRemoteRequiredOpts(env *lsf.Environment, args ...string) error {
	if e := verifyRequiredOption(updateRemoteOptions.id); e != nil {
		return e
	}
	return nil
}

func runUpdateRemote(env *lsf.Environment, args ...string) (err error) {
	panics.Recover(&err)

	id := *updateRemoteOptions.id.value
	updates := make(map[string][]byte)

	// update remote config document
	var option interface{}
	option = updateRemoteOptions.id
	if OptionProvided(option) {
		v := []byte(string(*updateRemoteOptions.id.value))
		updates[schema.PortElem.Id] = v
	}
	option = updateRemoteOptions.host
	if OptionProvided(option) {
		v := []byte(string(*updateRemoteOptions.host.value))
		updates[schema.PortElem.Host] = v
	}
	option = updateRemoteOptions.port
	if OptionProvided(option) {
		v := []byte(fmt.Sprintf("%d", (*updateRemoteOptions.port.value)))
		updates[schema.PortElem.PortNum] = v
	}

	return env.UpdateRemotePort(id, updates)
}
