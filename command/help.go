package command

import (
	"lsf"
)

const cmd_help lsf.CommandCode = "help"

var Help *lsf.Command
var helpOptions *helpOptionsSpec

type helpOptionsSpec struct {
	command StringOptionSpec
}

func init() {

	Help = &lsf.Command{
		Name:  cmd_help,
		About: "Provides usage information for LS/F commands",
		Run:   runHelp,
		Flag:  FlagSet(cmd_help),
		Usage: "help <command>",
	}

	helpOptions = &helpOptionsSpec{
		command: NewStringOptionSpec("c", "command", "", "the command you need help with", true),
	}
	helpOptions.command.defineFlag(Help.Flag)
}

func runHelp(env *lsf.Environment, args ...string) error {
	panic("command.help() not impelemented!")
}
