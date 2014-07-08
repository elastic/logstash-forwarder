package command

import (
	"github.com/elasticsearch/kriterium/panics"
	"lsf"
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

func runRemoveStream(env *lsf.Environment, args ...string) (err error) {
	panics.Recover(&err)

	id := *removeStreamOptions.id.value
	return env.RemoveLogStream(id)
}
