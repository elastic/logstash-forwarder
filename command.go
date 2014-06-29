package lsf

import (
	"flag"
	"lsf/panics"
	"os"
	"os/signal"
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
	IsActive    bool
}

// REVU: os signal trap should be here.
// REVU: fork on IsActive, keep 2 Run flavors.
// REVU: (non-critical) (os level) process fork the command runner
// Run the command. Trap any panics and return as error.
func Run(env *Environment, cmd *Command, args ...string) (err error) {

	defer panics.Recover(&err)

	// environment is created only if it is nil
	// AND command is not an initializer.
	// We set the env shutdown hook here
	if env == nil && !cmd.Initializer {
		env = NewEnvironment()
		e := env.Initialize(Wd())
		panics.OnError(e, "command.Run:", "env.Initialize:")
		defer func() {
			env.Shutdown()
		}()
	}

	cmd.Flag.Parse(args)
	commandArgs := cmd.Flag.Args()

	if cmd.IsActive {
		return RunActive(env, cmd, commandArgs...)
	}

	return RunPassive(env, cmd, commandArgs...)
}

func RunPassive(env *Environment, cmd *Command, args ...string) (err error) {

	defer panics.Recover(&err)

	// run cmd initializer func (if any)
	if cmd.Init != nil {
		e0 := cmd.Init(env, args...)
		panics.OnError(e0)
	}
	panics.OnTrue(cmd.Run == nil, "command.RunPassive:", "BUG - cmd.Run is nil")

	e1 := cmd.Run(env, args...)
	panics.OnError(e1)

	// run cmd finalizer func (if any)
	if cmd.End != nil {
		e2 := cmd.End(env, args...)
		panics.OnError(e2)
	}

	return nil
}

func RunActive(env *Environment, cmd *Command, args ...string) (err error) {

	defer panics.Recover(&err)

	if cmd.Init != nil {
		e0 := cmd.Init(env, args...)
		panics.OnError(e0)
	}
	panics.OnTrue(cmd.Run == nil, "command.RunActive:", "BUG - cmd.Run is nil")

	user := make(chan os.Signal, 1)
	user0 := make(chan os.Signal, 1)
	signal.Notify(user, os.Interrupt, os.Kill)
	signal.Notify(user0, os.Interrupt, os.Kill)

	prev, e := env.Set(VarUserSigChan, user)
	panics.OnError(e, "command.RunActive:", "BUG", "env.Set(VarUserSigChan)")
	panics.OnTrue(prev != nil, "command.RunActive:", "BUG", "env.Set(VarUserSigChan) returned non-nil value")

	e1 := cmd.Run(env, args...)
	panics.OnError(e1)

	// run cmd finalizer func (if any)
	if cmd.End != nil {
		e2 := cmd.End(env, args...)
		panics.OnError(e2)
	}

	return nil
}
