package main

import (
	"os"
	"fmt"
	"log"
	"flag"
	"github.com/elasticsearch/kriterium/panics"
	"lsf"
	"lsf/command"
)

type syserrCode int

const (
	e_none syserrCode = iota
	e_usage
	e_environment_init
	e_command_run
)

// List of available commands
var commands = []*lsf.Command{
	command.Debug,
	command.Help,
	command.Init,
	command.Stream,
	command.Remote,
	command.Track,
}

var debugFlag bool
var aboutDebug = "for development - disables panic recovery - use in case of ambiguous errors."

var aboutFlag bool
var aboutAbout = "provides conceptual background about LS/F."

var homeOption string
var aboutHome = "path to the lsf port"

// initialize command runner flag options
func init() {
	flag.BoolVar(&debugFlag, "debug", false, aboutDebug)
	flag.BoolVar(&aboutFlag, "about", false, aboutAbout)
	flag.StringVar(&homeOption, "home", getWd(), aboutHome)
}

// Command runner tool.
func main() {
	//	var err = fmt.Errorf("")
	//	defer panics.Recover(&err)
	log.SetFlags(0)

	// parse command runner flags

	flag.Parse()
	if aboutFlag {
		aboutLsfThenExit()
	}

	if debugFlag {
		panics.DEBUG = true
	}

	// first arg after runner's options must
	// be the command to run.

	args := flag.Args()
	if len(args) < 1 {
		usageThenExit()
	}

	// find and run the command

	cmdcode := lsf.CommandCode(args[0])
	cmd := getCommand(cmdcode)
	if cmd == nil {
		usageThenExit()
	}

	e := lsf.Run(nil, cmd, args[1:]...)
	if e != nil {
		onError(e, e_command_run)
	}

	return
}

func getWd() string {
	wd, e := os.Getwd()
	if e != nil {
		panic(e)
	}
	return wd
}

// Get command the identified instance
func getCommand(code lsf.CommandCode) *lsf.Command {
	for _, cmd := range commands {
		if cmd.Name == code {
			return cmd
		}
	}
	return nil
}

// system exits
func onError(e error, code syserrCode) {
	log.Printf("%s", e.Error())
	os.Exit(int(code))
}

// display usage of lsf and exit (usage error)
func usageThenExit() {
	var usage = `usage: lsf [-home <path>] <command> [<options>] [<args>]

Commands are:
`
	for _, cmd := range commands {
		usage += fmt.Sprintf("    %-12s%s\n", cmd.Name, cmd.About)
	}
	usage += `
lsf's environment is based on your current working directory.
Use 'lsf -home <lsf-port-path> ...' to use a specific lsf port.

'lsf help <command>' displays usage details for the command.
'lsf --about' ` + aboutAbout

	log.Print(usage)
	os.Exit(int(e_usage))
}

// the 'about' blurb
var aboutLsf = `Log Stash Forwarder
[TODO: LS/F general info ...]
`

// display the 'about' blurb and exit (OK)
func aboutLsfThenExit() {
	log.Print(aboutLsf)
	os.Exit(int(e_none))
}
