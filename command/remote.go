package command

import (
	"lsf"
)

const remoteCmdCode lsf.CommandCode = "remote"

type remoteOptionsSpec struct {
	verbose BoolOptionSpec
	global  BoolOptionSpec
}

var Remote *lsf.Command
var remoteOptions *remoteOptionsSpec

const (
	remoteOptionVerbose = "command.remote.option.verbose"
	remoteOptionGlobal  = "command.remote.option.global"
)

func init() {

	Remote = &lsf.Command{
		Name:  remoteCmdCode,
		About: "Remote is a top level command for remote specific features of LSF",
		//		Init:  initialCmdEnv,
		Run:  runRemote,
		Flag: FlagSet(remoteCmdCode),
	}
	remoteOptions = &remoteOptionsSpec{
		verbose: NewBoolFlag(Remote.Flag, "v", "verbose", false, "be verbose in list", false),
		global:  NewBoolFlag(Remote.Flag, "G", "global", false, "command applies globally", false),
	}
}

func runRemote(env *lsf.Environment, args ...string) error {

	if *remoteOptions.verbose.value {
		env.Set(remoteOptionVerbose, true)
	}
	if *remoteOptions.global.value {
		env.Set(remoteOptionGlobal, true)
	}

	xoff := 0
	var subcmd *lsf.Command = listRemote
	if len(args) > 0 {
		xoff = 1
		switch lsf.CommandCode("remote-" + args[0]) {
		case addRemoteCmdCode:
			subcmd = addRemote
		case removeRemoteCmdCode:
			subcmd = removeRemote
		case updateRemoteCmdCode:
			subcmd = updateRemote
		case listRemoteCmdCOde:
			subcmd = listRemote
		default:
			// not panic -- return error TODO
			panic("BUG - unknown subcommand for remote: " + args[0])
		}
	}

	return lsf.Run(env, subcmd, args[xoff:]...)
}
