package command

import (
	"errors"
	"flag"
	"lsf"
)

const cmd_stream lsf.CommandCode = "stream"

type streamOptionsSpec struct {
	verbose BoolOptionSpec
}

type editStreamOptionsSpec struct {
	verbose BoolOptionSpec
	global  BoolOptionSpec
	id      StringOptionSpec
	mode    StringOptionSpec
	path    StringOptionSpec
	pattern StringOptionSpec
}

func initEditStreamOptionsSpec(flagset *flag.FlagSet) *editStreamOptionsSpec {
	return &editStreamOptionsSpec{
		verbose: NewBoolFlag(flagset, "v", "verbose", false, "be verbose in list", false),
		global:  NewBoolFlag(flagset, "G", "global", false, "global scope flag for command", false),
		id:      NewStringFlag(flagset, "s", "stream-id", "", "unique identifier for stream", true),
		path:    NewStringFlag(flagset, "p", "path", "", "path to log files", true),
		mode:    NewStringFlag(flagset, "m", "journal-mode", "", "stream journaling mode (rotation|rollover)", false),
		pattern: NewStringFlag(flagset, "n", "name-pattern", "", "naming pattern of journaled log files", true),
	}
}
func verifyEditStreamRequiredOpts(env *lsf.Environment, args ...string) error {
	if e := verifyRequiredOption(addStreamOptions.id); e != nil {
		return e
	}
	if e := verifyRequiredOption(addStreamOptions.pattern); e != nil {
		return e
	}
	if e := verifyRequiredOption(addStreamOptions.path); e != nil {
		return e
	}
	if e := verifyRequiredOption(addStreamOptions.mode); e != nil {
		return e
	}

	mode := *addStreamOptions.mode.value
	switch mode {
	case "rollover", "rotation":
	default:
		return errors.New("option mode must be one {rollover, rotation}")
	}
	return nil
}

var Stream *lsf.Command
var streamOptions *streamOptionsSpec

const (
	streamOptionVerbose   = "command.stream.option.verbose"
	streamOptionGlobal    = "command.stream.option.global"
	streamOptionsSelected = "command.stream.option.selected"
)

func init() {

	Stream = &lsf.Command{
		Name:  cmd_stream,
		About: "Stream is a top level command for log stream configuration and management",
		Run:   runStream,
		Flag:  FlagSet(cmd_stream),
	}
	streamOptions = &streamOptionsSpec{
		verbose: NewBoolFlag(Stream.Flag, "v", "verbose", false, "be verbose in list", false),
	}
}

func runStream(env *lsf.Environment, args ...string) error {

	if *streamOptions.verbose.value {
		env.Set(streamOptionVerbose, true)
	}

	xoff := 0
	var subcmd *lsf.Command = listStream
	if len(args) > 0 {
		subcmd = getSubCommand(args[0])
		xoff = 1
	}

	return lsf.Run(env, subcmd, args[xoff:]...)
}

func getSubCommand(subcmd string) *lsf.Command {

	var cmd *lsf.Command
	switch lsf.CommandCode("stream-" + subcmd) {
	case addStreamCmdCode:
		cmd = addStream
	case removeStreamCmdCode:
		cmd = removeStream
	case updateStreamCmdCode:
		cmd = updateStream
	case listStreamCmdCode:
		cmd = listStream
	default:
		// not panic -- return error TODO
		panic("BUG - unknown subcommand for stream: " + subcmd)
	}
	return cmd
}
