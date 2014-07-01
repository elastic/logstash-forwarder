package command

import (
	"lsf"
	"lsf/panics"
	//	"lsf/schema"
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

	//	id := *removeRemoteOptions.id.value
	//	updates := make(map[string][]byte)
	//
	//	// update remote config document
	//	var option interface{}
	//	option = updateRemoteOptions.id
	//	if OptionProvided(option) {
	//		v := []byte(string(*updateRemoteOptions.id.value))
	//		updates[schema.PortElem.Id] = v
	//	}
	//	option = updateRemoteOptions.host
	//	if OptionProvided(option) {
	//		v := []byte(string(*updateRemoteOptions.path.value))
	//		updates[schema.LogStreamElem.BasePath] = v
	//	}
	//	option = updateRemoteOptions.port
	//	if OptionProvided(option) {
	//		v := []byte(schema.ToJournalModel(*updateRemoteOptions.mode.value))
	//		updates[schema.LogStreamElem.JournalModel] = v
	//	}

	panic("command.runUpdateRemote() not impelemented!")
}
