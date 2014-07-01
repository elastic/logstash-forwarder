package command

import (
	"lsf"
	"lsf/panics"
)

const addStreamCmdCode lsf.CommandCode = "stream-add"

var addStream *lsf.Command
var addStreamOptions *editStreamOptionsSpec

func init() {

	addStream = &lsf.Command{
		Name:  addStreamCmdCode,
		About: "Add a new log stream",
		Init:  _verifyEditStreamRequiredOpts,
		Run:   runAddStream,
		Flag:  FlagSet(addStreamCmdCode),
	}
	addStreamOptions = initEditStreamOptionsSpec(addStream.Flag)
}

func runAddStream(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)

	id := *addStreamOptions.id.value
	pattern := *addStreamOptions.pattern.value
	journalMode := *addStreamOptions.mode.value
	basepath := *addStreamOptions.path.value
	fields := make(map[string]string) // TODO: fields needs a solution

	return env.AddLogStream(id, basepath, pattern, journalMode, fields)
}
