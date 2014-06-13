package command

import (
	"fmt"
	"lsf"
	"lsf/schema"
	"lsf/system"
	"lsf/anomaly"
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
		Init:  _verifyAddRemoteRequiredOpts,
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

func _verifyAddRemoteRequiredOpts(env *lsf.Environment, args ...string) (err error) {

	var e error
	e = verifyRequiredOption(addRemoteOptions.id)
	anomaly.PanicOnError(e, "remote-add", "option", "id")

	e = verifyRequiredOption(addRemoteOptions.host)
	anomaly.PanicOnError(e, "remote-add", "option", "host")

	e = verifyRequiredOption(addRemoteOptions.port)
	anomaly.PanicOnError(e, "remote-add", "option", "port")

	return
}

func runAddRemote(env *lsf.Environment, args ...string) (err error) {
	defer anomaly.Recover(&err)

	id := schema.StreamId(*addRemoteOptions.id.value)

	// check if exists
	docid := system.DocId(fmt.Sprintf("remote.%s.remote", id))
	_assertNotExists(env, docid)

	// lock lsf port's "remotes" resource
	// to prevent race condition
	lock := _lockResource(env, "remotes", "add remote port")
	defer lock.Unlock()

	panic("finish me")
}
