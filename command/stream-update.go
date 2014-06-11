package command

import (
	"fmt"
	"lsf"
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

func runUpdateStream(env *lsf.Environment, args ...string) error {

	if e := verifyRequiredOption(updateStreamOptions.id); e != nil {
		return e
	}
	id := schema.StreamId(*updateStreamOptions.id.value)

	// do not premit concurrent updates to this stream
	lockid := env.ResourceId(fmt.Sprintf("stream.%s.update", id))
	oplock, ok, e := system.LockResource(lockid, "add stream "+string(id))
	if e != nil {
		return e
	}
	if !ok {
		return fmt.Errorf("error - could not lock resource %q for stream update op", string(id))
	}
	defer oplock.Unlock()

	// verify it exists
	docid := system.DocId(fmt.Sprintf("stream.%s.stream", id))
	doc, e := env.LoadDocument(docid)
	if e != nil || doc == nil {
		panic("BUG - error or document for stream missing: " + docid)
	}

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
		v := []byte(schema.JournalModel(*updateStreamOptions.mode.value))
		doc.Set("journal-mode", v)
	}
	ok, e = env.UpdateDocument(doc)
	if e != nil {
		return fmt.Errorf("error runUpdateStream: %s", e)
	}
	if !ok {
		return fmt.Errorf("error runUpdateStream: UpdateDocument returned false")
	}

	return nil
}
