package lsf

import (
	"flag"
	"fmt"
	"github.com/elasticsearch/kriterium/panics"
	"log"
	"lsf/system"
	"lsf/system/process"
	"os"
	"os/signal"
)

// REVU: no errors? TODO: consolidate all errors under lsf/errors
var Status = struct {
	Ok, Interrupted_Ok /*, Faulted*/ string
}{
	Ok:             "ok",
	Interrupted_Ok: "Interrupted_Ok",
	//	Faulted:        "Faulted",
}

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
	process     system.Process
}

//type ActiveCommand struct {
//	*Command
//	*process.Control
//}

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

func RunActive(env *Environment, cmd0 *Command, args ...string) (err error) {

	defer panics.Recover(&err)

	procControl := process.NewProcessControl()
	env.Set(VarSupervisor, system.Supervisor(procControl))

	if cmd0.Init != nil {
		e0 := cmd0.Init(env, args...)
		panics.OnError(e0)
	}
	panics.OnTrue(cmd0.Run == nil, "command.RunActive:", "BUG - cmd.Run is nil")

	var cmd = struct {
		cmd  *Command
		proc *process.Control
	}{
		cmd:  cmd0,
		proc: procControl,
	}

	user := make(chan os.Signal, 1)
	signal.Notify(user, os.Interrupt, os.Kill)

	// expected to run a go routine
	// this call should NOT block
	// REVU: and why shouldn't it block?
	// TODO: REVU alt approach.
	e1 := cmd.cmd.Run(env, args...)
	panics.OnError(e1)

	// act as command process supervisor
	//
	var stat interface{}
	select {
	case stat = <-cmd.proc.Status():
		// For now any stat message is read as 'done'.
		// Here it can only mean that either the task
		// inherently is done OR it encountered an error.
		break
	case usersig := <-user:
		// stop the command process on user signal
		cmd.proc.Signal() <- usersig
		stat = <-cmd.proc.Status()
		break
	}

	// TODO: act on 'stat'
	switch stat {
	case Status.Ok, Status.Interrupted_Ok:
		log.Printf("\nstream-track: %v", stat)
	default:
		err = fmt.Errorf("track fault on exit: %v", stat)
		log.Printf("\n%s", err.Error())
	}

	// run cmd finalizer func (if any)
	if cmd.cmd.End != nil {
		e2 := cmd.cmd.End(env, args...)
		panics.OnError(e2)
	}

	return
}
