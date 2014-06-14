package command

import (
	"log"
	"lsf"
	"lsf/schema"
	"lsf/anomaly"
)

const cmd_remote_list lsf.CommandCode = "remote-list"

type listRemoteOptionsSpec struct {
	global  BoolOptionSpec
	verbose BoolOptionSpec
}

var listRemote *lsf.Command
var listRemoteOptions *listRemoteOptionsSpec

func init() {

	listRemote = &lsf.Command{
		Name:  cmd_remote_list,
		About: "List Remotes defined ",
		Run:   runListRemote,
		Flag:  FlagSet(cmd_remote_list),
	}
	listRemoteOptions = &listRemoteOptionsSpec{
		global:  NewBoolFlag(listRemote.Flag, "G", "global", false, "global scope flag for command", false),
		verbose: NewBoolFlag(listRemote.Flag, "v", "verbose", false, "detailed output", false),
	}
}

func runListRemote(env *lsf.Environment, args ...string) (err error) {
	defer anomaly.Recover(&err)

	verbose := *listRemoteOptions.verbose.value
	v, found := env.Get(remoteOptionVerbose)
	if found {
		verbose = verbose || v.(bool)
	}

	digests := env.GetResourceDigests("remote", verbose, schema.PortDigest)
	for _, digest := range digests {
		log.Println(digest)
	}

	return nil
}
