package command

import (
	"fmt"
	"lsf"
	"lsf/panics"
	"lsf/system"
	"os"
)

const removeStreamCmdCode lsf.CommandCode = "stream-remove"

type removeStreamOptionsSpec struct {
	global BoolOptionSpec
	id     StringOptionSpec
}

var removeStream *lsf.Command
var removeStreamOptions *removeStreamOptionsSpec

func init() {

	removeStream = &lsf.Command{
		Name:  removeStreamCmdCode,
		About: "Remove a new log stream",
		Init:  verifyRemoveStreamRequiredOpts,
		Run:   runRemoveStream,
		Flag:  FlagSet(removeStreamCmdCode),
	}
	removeStreamOptions = &removeStreamOptionsSpec{
		global: NewBoolFlag(removeStream.Flag, "G", "global", false, "global scope operation", false),
		id:     NewStringFlag(removeStream.Flag, "s", "stream-id", "", "unique identifier for stream", true),
	}
}
func verifyRemoveStreamRequiredOpts(env *lsf.Environment, args ...string) error {
	if e := verifyRequiredOption(removeStreamOptions.id); e != nil {
		return e
	}
	return nil
}

// REVU: TODO definitively require a stream 'x' lock for use by
// processes that expect the stream (info) to remain in place.
// For now, assuming this is the same "stream.<name>.stream.lock"
// lock file.
func runRemoveStream(env *lsf.Environment, args ...string) (err error) {
	panics.Recover(&err)

	id := *removeStreamOptions.id.value

	// check existing
	docId := fmt.Sprintf("stream.%s.stream", id)
	doc, e := env.LoadDocument(docId)
	if e != nil || doc == nil {
		return lsf.E_NOTEXISTING
	}

	// lock lsf port's "streams" resource
	lockid := env.ResourceId("streams")
	lock, ok, e := system.LockResource(lockid, "add stream "+id)
	panics.OnError(e, "command.runRemoveStream:", "lockResource:")
	panics.OnFalse(ok, "command.runRemoveStream:", "lockResource:", id)
	defer lock.Unlock()

	// remove doc
	ok, e = env.DeleteDocument(docId)
	panics.OnError(e, "command.runRemoveStream:", "DeleteDocument:", id)
	panics.OnFalse(ok, "command.runRemoveStream:", "DeleteDocument:", id)

	// remove the stream's directory
	// REVU: this command needs a check to see if any procs
	// related to this stream are running . OK for initial.
	dir, fname := system.DocpathForKey(env.Port(), docId)
	fmt.Printf("DEBUG: runRemoveStream: %s %s\n", dir, fname)

	e = os.RemoveAll(dir)
	panics.OnError(e, "command.runRemoveStream:", "os.RemoveAll:", dir)

	return nil
}
