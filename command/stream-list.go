package command

import (
	"fmt"
	"log"
	"lsf"
	"lsf/schema"
	"lsf/system"
	"lsf/anomaly"
	"os"
	"path"
)

const listStreamCmdCode lsf.CommandCode = "stream-list"

type listStreamOptionsSpec struct {
	global  BoolOptionSpec
	verbose BoolOptionSpec
}

var listStream *lsf.Command
var listStreamOptions *listStreamOptionsSpec

func init() {

	listStream = &lsf.Command{
		Name:  listStreamCmdCode,
		About: "List Streams defined ",
		Run:   runListStream,
		Flag:  FlagSet(listStreamCmdCode),
	}
	listStreamOptions = &listStreamOptionsSpec{
		global:  NewBoolFlag(listStream.Flag, "G", "global", false, "global scope flag for command", false),
		verbose: NewBoolFlag(listStream.Flag, "v", "verbose", false, "detailed output", false),
	}
}

func runListStream(env *lsf.Environment, args ...string) (err error) {
	defer anomaly.Recover(&err)

	//	global := *listStreamOptions.global.value

	verbose := *listStreamOptions.verbose.value
	v, found := env.Get(streamOptionVerbose)
	if found {
		verbose = verbose || v.(bool)
	}

	root := env.Port()
	dir, e := os.Open(path.Join(root, "stream"))
	if e != nil {
		return nil // no stream dir - nothing to list
	}

	dirnames, e := dir.Readdirnames(0)
	anomaly.PanicOnError(e, "runListStream:", "BUG - directory is empty", dir.Name())

	for _, sid := range dirnames {
		if sid[0] == '.' {
			continue
		}
		info := sid
		if verbose {
			docid := system.DocId(fmt.Sprintf("stream.%s.stream", sid))
			doc, e := env.LoadDocument(docid)
			anomaly.PanicOnError(e, "runListStream:", "env.LoadDocument:", "BUG")
			anomaly.PanicOnTrue(doc == nil, "runListStream:", "env.LoadDocument:", "BUG")

			logstream := schema.DecodeLogStream(doc)
			log.Printf("%s", logstream.String())
		} else {
			log.Printf("%s", info)
		}
	}

	return nil
}
