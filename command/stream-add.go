package command

import (
	"fmt"
	"lsf"
	"lsf/schema"
	"lsf/system"
	"lsf/anomaly"
)

const addStreamCmdCode lsf.CommandCode = "stream-add"

var addStream *lsf.Command
var addStreamOptions *editStreamOptionsSpec

func init() {

	addStream = &lsf.Command{
		Name:  addStreamCmdCode,
		About: "Add a new log stream",
		Init:  verifyEditStreamRequiredOpts,
		Run:   runAddStream,
		Flag:  FlagSet(addStreamCmdCode),
	}
	addStreamOptions = initEditStreamOptionsSpec(addStream.Flag)
}

func runAddStream(env *lsf.Environment, args ...string) (err error) {
	defer anomaly.Recover(&err)

	id := schema.StreamId(*addStreamOptions.id.value)
	pattern := *addStreamOptions.pattern.value
	mode := schema.JournalModel(*addStreamOptions.mode.value)
	path := *addStreamOptions.path.value
	fields := make(map[string]string) // TODO: fields needs a solution

	// check existing
	docid := system.DocId(fmt.Sprintf("stream.%s.stream", id))
	doc, e := env.LoadDocument(docid)
	if e == nil && doc != nil {
		return lsf.E_EXISTING
	}

	// lock lsf port's "streams" resource to prevent race condition
	lockid := env.ResourceId("streams")
	lock, ok, e := system.LockResource(lockid, "add stream "+string(id))
	anomaly.PanicOnError(e, "command.runAddStream:", "lockResource:")
	anomaly.PanicOnFalse(ok, "command.runAddStream:", "lockResource:", string(id))
	defer lock.Unlock()

	// create the stream-conf file.
	logstream := schema.NewLogStream(id, path, mode, pattern, fields)
	e = env.CreateDocument(docid, logstream)
	anomaly.PanicOnError(e, "command.runAddStream:", "CreateDocument:", string(id))

	return nil
}
