package lsf

import (
	"flag"
	"fmt"
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
	defer func() {
		if p := recover(); p != nil {
			var estr interface{}
			switch t := p.(type) {
			case error:
				estr = t.Error()
			case string:
				estr = p
			default:
				estr = p // what could it be? has to be lsf
			}
			err = fmt.Errorf("lsf.Run: %s: recovered panic: %s", cmd.Name, estr)
		}
	}()

	// environment is created only if it is nil
	// AND command is not an initializer.
	// We set the env shutdown hook here
	if env == nil && !cmd.Initializer {
		env = NewEnvironment()
		e := env.Initialize(Wd())
		if e != nil {
			return fmt.Errorf("error: lsf.Run: initializing environment: %s", e)
		}
		defer func() {
			env.Shutdown()
		}()
	}

	cmd.Flag.Parse(args)
	commandArgs := cmd.Flag.Args()
	if cmd.Init != nil {
		e0 := cmd.Init(env, commandArgs...)
		if e0 != nil {
			return fmt.Errorf("(init) %s: %s", cmd.Name, e0)
		}
	}
	if cmd.Run == nil {
		panic(fmt.Errorf("BUG - %s.Run is nil", cmd.Name))
	}
	e1 := cmd.Run(env, commandArgs...)
	if e1 != nil {
		return fmt.Errorf("(run) %s: %s", cmd.Name, e1)
	}
	if cmd.End != nil {
		e2 := cmd.End(env, commandArgs...)
		if e2 != nil {
			return fmt.Errorf("(End) %s: %s", cmd.Name, e2)
		}
	}

	return nil
}
