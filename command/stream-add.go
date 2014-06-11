package command

import (
	"fmt"
	"lsf"
	"lsf/schema"
	"lsf/system"
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

func runAddStream(env *lsf.Environment, args ...string) error {

	id := schema.StreamId(*addStreamOptions.id.value)
	pattern := *addStreamOptions.pattern.value
	mode := schema.JournalModel(*addStreamOptions.mode.value)
	path := *addStreamOptions.path.value
	fields := make(map[string]string) // TODO: fields needs a solution

	// check if exists
	docid := system.DocId(fmt.Sprintf("stream.%s.stream", id))
	doc, e := env.LoadDocument(docid)
	if e == nil && doc != nil {
		return lsf.E_EXISTING
	}

	// lock lsf port's "streams" resource
	// to prevent race condition
	lockid := env.ResourceId("streams")
	//	log.Printf("DEBUG: runAddStream: lockid: %q", lockid)
	lock, ok, e := system.LockResource(lockid, "add stream "+string(id))
	if e != nil {
		return e
	}
	if !ok {
		return fmt.Errorf("error - could not lock resource %q for stream add op", string(id))
	}
	defer lock.Unlock()

	// create the stream-conf file.
	logstream := schema.NewLogStream(id, path, mode, pattern, fields)
	e = env.CreateDocument(docid, logstream)
	if e != nil {
		return e
	}

	return nil
}
