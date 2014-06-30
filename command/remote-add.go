package command

import (
	"fmt"
	"lsf"
	"lsf/panics"
	"lsf/schema"
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
		id:   NewStringFlag(addRemote.Flag, "r", "remote-id", "", "unique identifier for remote port", true),
		host: NewStringFlag(addRemote.Flag, "h", "remote-host", "", "URL of the remote port", true),
		port: NewInt64Flag(addRemote.Flag, "p", "remote-port", 0, "IP port number of remote port", true),
	}
}

func _verifyAddRemoteRequiredOpts(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)

	var e error
	e = verifyRequiredOption(addRemoteOptions.id)
	panics.OnError(e, "remote-add", "option", "id")

	e = verifyRequiredOption(addRemoteOptions.host)
	panics.OnError(e, "remote-add", "option", "host")

	e = verifyRequiredOption(addRemoteOptions.port)
	panics.OnError(e, "remote-add", "option", "port")

	return
}

func runAddRemote(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)

	id := *addRemoteOptions.id.value
	host := *addRemoteOptions.host.value
	port := int(*addRemoteOptions.port.value)

	// check if exists
	docId := fmt.Sprintf("remote.%s.remote", id)
	_assertNotExists(env, docId)

	// lock lsf port's "remotes" resource to prevent race condition
	lock := _lockResource(env, "remotes", "add remote port")
	defer lock.Unlock()

	lsfport, e := schema.NewRemotePort(id, host, port)
	panics.OnError(e, "runAddRemote:", "NewRemotePort")

	e = env.CreateDocument(docId, lsfport)
	panics.OnError(e, "command.runAddStream:", "CreateDocument:", id)

	return nil
}
