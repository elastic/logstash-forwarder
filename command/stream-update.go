package command

import (
	"fmt"
	"lsf"
	"lsf/panics"
	"lsf/schema"
	"lsf/system"
)

const updateStreamCmdCode lsf.CommandCode = "stream-update"

//type updateStreamOptionsSpec struct {
//	global BoolOptionSpec
//}

var updateStream *lsf.Command
var updateStreamOptions *editStreamOptionsSpec

func init() {

	updateStream = &lsf.Command{
		Name:  updateStreamCmdCode,
		About: "Update a new log stream",
		Run:   runUpdateStream,
		Flag:  FlagSet(updateStreamCmdCode),
	}
	updateStreamOptions = initEditStreamOptionsSpec(updateStream.Flag)
}

func runUpdateStream(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)

	e := verifyRequiredOption(updateStreamOptions.id)
	panics.OnError(e, "runUpdateStream:", "verifyRequiredOption")

	id := *updateStreamOptions.id.value

	// do not premit concurrent updates to this stream
	opLock, _, e := system.ExclusiveResourceOp(env.Port(), system.Op.StreamUpdate, id, "stream-update command")
	panics.OnError(e, "system.Op.StreamUpdate")
	defer opLock.Unlock()

	// verify it exists
	docId := fmt.Sprintf("stream.%s.stream", id)
	doc, e := env.LoadDocument(docId)
	panics.OnError(e, "BUG command.runUpdateStream:", "LoadDocument:", docId)
	panics.OnTrue(doc == nil, "BUG command.runUpdateStream:", "LoadDocument:", docId)

	// update stream config document
	var option interface{}
	option = updateStreamOptions.pattern
	if OptionProvided(option) {
		v := []byte(string(*updateStreamOptions.pattern.value))
		doc.Set(schema.LogStreamElem.Pattern, v)
	}
	option = updateStreamOptions.path
	if OptionProvided(option) {
		v := []byte(string(*updateStreamOptions.path.value))
		doc.Set(schema.LogStreamElem.BasePath, v)
	}
	option = updateStreamOptions.mode
	if OptionProvided(option) {
		v := []byte(schema.ToJournalModel(*updateStreamOptions.mode.value))
		doc.Set(schema.LogStreamElem.JournalModel, v)
	}
	ok, e := env.UpdateDocument(doc)
	panics.OnError(e, "command.runUpdateStream:", "UpdateDocument:", id)
	panics.OnFalse(ok, "command.runUpdateStream:", "UpdateDocument:", id)

	return nil
}
