package lsf

import (
	"flag"
	"lsf/anomaly"
)

// REVU: no errors? TODO: consolidate all errors under lsf/errors

// lsf Command function
//type CommandFn func(context map[string]interface{}, args ...string)
type CommandFn func(env *Environment, args ...string) error
type CommandInitFn func(env *Environment, args ...string) error
type CommandEndFn func(env *Environment, args ...string) error

// unique name of command
type CommandCode string

// Possibly easier to type string(t) .. ?
func (t CommandCode) String() string {
	return string(t)
}

// An LSF Command descriptor
type Command struct {
	Name        CommandCode
	About       string
	Init        CommandInitFn
	Run         CommandFn
	End         CommandEndFn
	Flag        *flag.FlagSet
	Usage       string
	Initializer bool
}

// Run the command. Trap any panics and return as error.
func Run(env *Environment, cmd *Command, args ...string) (err error) {

	defer anomaly.Recover(&err)

	// environment is created only if it is nil
	// AND command is not an initializer.
	// We set the env shutdown hook here
	if env == nil && !cmd.Initializer {
		env = NewEnvironment()
		e := env.Initialize(Wd())
		anomaly.PanicOnError(e, "command.Run:", "env.Initialize:")
		defer func() {
			env.Shutdown()
		}()
	}

	cmd.Flag.Parse(args)
	commandArgs := cmd.Flag.Args()

	// run cmd initializer func (if any)
	if cmd.Init != nil {
		e0 := cmd.Init(env, commandArgs...)
		anomaly.PanicOnError(e0)
		//		anomaly.PanicOnError(e0, "command.Run:", cmd.Name.String(), "Init()")
	}

	// treat nil cmd.Run as bug
	anomaly.PanicOnTrue(cmd.Run == nil, "command.Run:", "BUG - cmd.Run is nil")

	e1 := cmd.Run(env, commandArgs...)
	anomaly.PanicOnError(e1)
	//	anomaly.PanicOnError(e1, "command.Run:", cmd.Name.String(), "Run()")

	// run cmd finalizer func (if any)
	if cmd.End != nil {
		e2 := cmd.End(env, commandArgs...)
		anomaly.PanicOnError(e2)
		//		anomaly.PanicOnError(e2, "command.Run:", cmd.Name.String(), "End()")
	}

	return nil
}
