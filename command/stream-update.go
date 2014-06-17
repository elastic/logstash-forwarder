package command

import (
	"fmt"
	"lsf"
	"lsf/anomaly"
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
	defer anomaly.Recover(&err)

	e := verifyRequiredOption(updateStreamOptions.id)
	anomaly.PanicOnError(e, "runUpdateStream:", "verifyRequiredOption")

	id := schema.StreamId(*updateStreamOptions.id.value)

	// do not premit concurrent updates to this stream
	resource := fmt.Sprintf("stream.%s.update", id)
	lockid := env.ResourceId(resource)
	oplock, ok, e := system.LockResource(lockid, "add stream - resource "+resource)
	anomaly.PanicOnError(e, "command.runUpdateStream:", "lockResource:", resource)
	anomaly.PanicOnFalse(ok, "command.runUpdateStream:", "lockResource:", resource)
	defer oplock.Unlock()

	// verify it exists
	docid := system.DocId(fmt.Sprintf("stream.%s.stream", id))
	doc, e := env.LoadDocument(docid)
	anomaly.PanicOnError(e, "BUG command.runUpdateStream:", "LoadDocument:", string(docid))
	anomaly.PanicOnTrue(doc == nil, "BUG command.runUpdateStream:", "LoadDocument:", string(docid))

	// update stream config document
	var option interface{}
	option = updateStreamOptions.pattern
	if OptionProvided(option) {
		v := []byte(string(*updateStreamOptions.pattern.value))
		doc.Set("name-pattern", v)
	}
	option = updateStreamOptions.path
	if OptionProvided(option) {
		v := []byte(string(*updateStreamOptions.path.value))
		doc.Set("file-path", v)
	}
	option = updateStreamOptions.mode
	if OptionProvided(option) {
		v := []byte(schema.ToJournalModel(*updateStreamOptions.mode.value))
		doc.Set("journal-mode", v)
	}
	ok, e = env.UpdateDocument(doc)
	anomaly.PanicOnError(e, "command.runUpdateStream:", "UpdateDocument:", resource)
	anomaly.PanicOnFalse(ok, "command.runUpdateStream:", "UpdateDocument:", resource)

	return nil
}
