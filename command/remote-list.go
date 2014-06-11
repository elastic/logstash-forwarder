package command

import (
	"fmt"
	"log"
	"lsf"
	"lsf/schema"
	"lsf/system"
	"os"
	"path"
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

func runListRemote(env *lsf.Environment, args ...string) error {

	//	global := *listRemoteOptions.global.value

	verbose := *listRemoteOptions.verbose.value
	v, found := env.Get(remoteOptionVerbose)
	if found {
		verbose = verbose || v.(bool)
	}

	root := env.Port()
	dir, e := os.Open(path.Join(root, "remote"))
	if e != nil {
		return nil // no remote dir - nothing to list
	}
	dirnames, e := dir.Readdirnames(0)
	if e != nil {
		return e
	}
	for _, sid := range dirnames {
		if sid[0] == '.' {
			continue
		}
		info := sid
		if verbose {
			docid := system.DocId(fmt.Sprintf("remote.%s", sid))
			doc, e := env.LoadDocument(docid)
			if e != nil || doc == nil {
				panic("BUG - error or document for remote missing: " + docid)
			}
			logremote := schema.DecodeLogStream(doc)
			log.Printf("%s", logremote.String())
		} else {
			log.Printf("%s", info)
		}
	}

	return nil
}
