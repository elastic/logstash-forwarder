package command

import (
	"github.com/elasticsearch/kriterium/panics"
	"log"
	"lsf"
	"lsf/schema"
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
	defer panics.Recover(&err)

	verbose := *listStreamOptions.verbose.value
	v, found := env.Get(streamOptionVerbose)
	if found {
		verbose = verbose || v.(bool)
	}

	digests := env.GetResourceDigests("stream", verbose, schema.LogStreamDigest)
	for _, digest := range digests {
		log.Println(digest)
	}

	return nil
}
