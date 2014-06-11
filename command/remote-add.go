package command

import (
	"fmt"
	"lsf"
	"lsf/schema"
	"lsf/system"
)

const addRemoteCmdCode lsf.CommandCode = "remote-add"

type addRemoteOptionsSpec struct {
	global BoolOptionSpec
	id     StringOptionSpec
	host   StringOptionSpec
	port   Int64OptionSpec
}

var addRemote *lsf.Command
var addRemoteOptions *addRemoteOptionsSpec

func init() {

	addRemote = &lsf.Command{
		Name:  addRemoteCmdCode,
		About: "Add a new log remote",
		Init:  verifyAddRemoteRequiredOpts,
		Run:   runAddRemote,
		Flag:  FlagSet(addRemoteCmdCode),
	}
	addRemoteOptions = &addRemoteOptionsSpec{
		//		global: NewBoolFlag(addRemote.Flag, "G", "global", false, "global scope flag for command", false),
		id:   NewStringFlag(addRemote.Flag, "s", "remote-id", "", "unique identifier for remote port", true),
		host: NewStringFlag(addRemote.Flag, "h", "remote-host", "", "URL of the remote port", true),
		port: NewInt64Flag(addRemote.Flag, "p", "remote-port", 0, "IP port number of remote port", true),
	}
}

func verifyAddRemoteRequiredOpts(env *lsf.Environment, args ...string) error {
	if e := verifyRequiredOption(addRemoteOptions.id); e != nil {
		return e
	}
	if e := verifyRequiredOption(addRemoteOptions.host); e != nil {
		return e
	}
	if e := verifyRequiredOption(addRemoteOptions.port); e != nil {
		return e
	}

	return nil
}

func runAddRemote(env *lsf.Environment, args ...string) error {

	id := schema.StreamId(*addRemoteOptions.id.value)
	//	pattern := *addRemoteOptions.pattern.value
	//	mode := schema.JournalModel(*addRemoteOptions.mode.value)
	//	path := *addRemoteOptions.path.value
	//	fields := make(map[string]string) // TODO: fields needs a solution

	// check if exists
	docid := system.DocId(fmt.Sprintf("remote.%s.remote", id))
	doc, e := env.LoadDocument(docid)
	if e == nil && doc != nil {
		return lsf.E_EXISTING
	}

	// lock lsf port's "remotes" resource
	// to prevent race condition
	lockid := env.ResourceId("remotes")
	//	log.Printf("DEBUG: runAddRemote: lockid: %q", lockid)
	lock, ok, e := system.LockResource(lockid, "add remote "+string(id))
	if e != nil {
		return e
	}
	if !ok {
		return fmt.Errorf("error - could not lock resource %q for remote add op", string(id))
	}
	defer lock.Unlock()

	// create the remote-conf file.
	//	logremote := schema.NewLogStream(id, path, mode, pattern, fields)
	//	e = env.CreateDocument(docid, logremote)
	//	if e != nil {
	//		return e
	//	}
	//
	//	return nil
	panic("finish me")
}
