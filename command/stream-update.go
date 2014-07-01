package command

import (
	"lsf"
	"lsf/panics"
	"lsf/schema"
)

const updateStreamCmdCode lsf.CommandCode = "stream-update"

var updateStream *lsf.Command
var updateStreamOptions *editStreamOptionsSpec

func init() {

	updateStream = &lsf.Command{
		Name:  updateStreamCmdCode,
		About: "Update a new log stream",
		Init:  verifyUpdateStreamRequiredOpts,
		Run:   runUpdateStream,
		Flag:  FlagSet(updateStreamCmdCode),
	}
	updateStreamOptions = initEditStreamOptionsSpec(updateStream.Flag)
}

func verifyUpdateStreamRequiredOpts(env *lsf.Environment, args ...string) error {
	if e := verifyRequiredOption(updateStreamOptions.id); e != nil {
		return e
	}
	return nil
}

func runUpdateStream(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)

	id := *updateStreamOptions.id.value
	updates := make(map[string][]byte)

	// update stream config document
	var option interface{}
	option = updateStreamOptions.pattern
	if OptionProvided(option) {
		v := []byte(string(*updateStreamOptions.pattern.value))
		updates[schema.LogStreamElem.Pattern] = v
	}
	option = updateStreamOptions.path
	if OptionProvided(option) {
		v := []byte(string(*updateStreamOptions.path.value))
		updates[schema.LogStreamElem.BasePath] = v
	}
	option = updateStreamOptions.mode
	if OptionProvided(option) {
		v := []byte(schema.ToJournalModel(*updateStreamOptions.mode.value))
		updates[schema.LogStreamElem.JournalModel] = v
	}

	return env.UpdateLogStream(id, updates)

}
