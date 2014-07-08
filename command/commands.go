package command

import (
	"flag"
	"fmt"
	"github.com/elasticsearch/kriterium/panics"
	"log"
	"lsf"
	"lsf/system"
)

// nil CommandFn is a nop function
//var nil_CommandFn CommandFn = func(context map[string]interface{}, args ...string) {}
var nil_CommandFn lsf.CommandFn = func(env *lsf.Environment, args ...string) error { return nil }
var nil_CommandInitFn lsf.CommandInitFn = func(env *lsf.Environment, args ...string) error { return nil }
var nil_CommandEndFn lsf.CommandEndFn = func(env *lsf.Environment, args ...string) error { return nil }

// nil Command has code nil_CommandCode and applies nil_CommandFn
var nil_command *lsf.Command = &lsf.Command{
	Name:  "",
	About: "",
	Init:  nil_CommandInitFn,
	Run:   nil_CommandFn,
	End:   nil_CommandEndFn,
	Flag:  nil,
	Usage: "",
}

// -----------------------------------------------------------------------
// runtime environment & command runner
// -----------------------------------------------------------------------

// panics on error
func getSupervisor(env *lsf.Environment) system.Supervisor {
	v, found := env.Get(lsf.VarSupervisor)
	panics.OnFalse(found, "BUG", "Get lsf.VarSupervisor")
	supervisor, ok := v.(system.Supervisor)
	panics.OnFalse(ok, "BUG", "Cast lsf.VarSupervisor")

	return supervisor
}

// Go process global scope, runtime environment for all commands
//var env *lsf.Environment = lsf.NewEnvironment() // not initialized ..

func debugArgs(args ...string) {
	log.Println("debug - args:")
	for i, arg := range args {
		log.Printf("[%02d] %s", i, arg)
	}
}

func initialCmdEnv(args ...string) error {
	panic("do it!")
	//	return env.Initialize(lsf.Wd())
}

func saveEnv(env *lsf.Environment, args ...string) error {
	panic("commands.go saveEnv: implement me!")
}

// -----------------------------------------------------------------------
// command line options and flags
// -----------------------------------------------------------------------
type OptionSpec struct {
	short, long, usage string
	required           bool
}

type RequiredOption interface {
	Provided() (ok bool)
}

func OptionProvided(opt interface{}) bool {
	return opt.(RequiredOption).Provided()
}

type FlagSpec interface {
	defineFlag(fs *flag.FlagSet)
}

// panics
func verifyRequiredOptions(options []interface{}) error {
	for _, option := range options {
		e := verifyRequiredOption(option)
		if e != nil {
			return e
		}
	}
	return nil
}

func verifyRequiredOption(option interface{}) error {

	if !option.(RequiredOption).Provided() {
		switch t := option.(type) {
		case StringOptionSpec:
			return fmt.Errorf("option '%s'|'%s' must be provided", t.long, t.short)
		case BoolOptionSpec:
			return fmt.Errorf("option '%s'|'%s' must be provided", t.long, t.short)
		case Int64OptionSpec:
			return fmt.Errorf("option '%s'|'%s' must be provided", t.long, t.short)
		case UintOptionSpec:
			return fmt.Errorf("option '%s'|'%s' must be provided", t.long, t.short)
		}
	}
	return nil
}

func NewOptionSpec(short, long, usage string, required bool) OptionSpec {
	return OptionSpec{short, long, usage, required}
}

type StringOptionSpec struct {
	OptionSpec
	value  *string
	defval string
}

func (opt StringOptionSpec) Provided() bool {
	return *opt.value != opt.defval
}

type Int64OptionSpec struct {
	OptionSpec
	value  *int64
	defval int64
}

func (opt Int64OptionSpec) Provided() bool {
	return *opt.value != opt.defval
}

type UintOptionSpec struct {
	OptionSpec
	value  *uint
	defval uint
}

func (opt UintOptionSpec) Provided() bool {
	return *opt.value != opt.defval
}

type BoolOptionSpec struct {
	OptionSpec
	value  *bool
	defval bool
}

func (opt BoolOptionSpec) Provided() bool {
	return *opt.value != opt.defval
}

func (t StringOptionSpec) defineFlag(fs *flag.FlagSet) {
	fs.StringVar(t.value, t.short, t.defval, t.usage)
	fs.StringVar(t.value, t.long, t.defval, t.usage)
}

func (t Int64OptionSpec) defineFlag(fs *flag.FlagSet) {
	fs.Int64Var(t.value, t.short, t.defval, t.usage)
	fs.Int64Var(t.value, t.long, t.defval, t.usage)
}

func (t UintOptionSpec) defineFlag(fs *flag.FlagSet) {
	fs.UintVar(t.value, t.short, t.defval, t.usage)
	fs.UintVar(t.value, t.long, t.defval, t.usage)
}

func (t BoolOptionSpec) defineFlag(fs *flag.FlagSet) {
	fs.BoolVar(t.value, t.short, t.defval, t.usage)
	fs.BoolVar(t.value, t.long, t.defval, t.usage)
}

func NewStringOptionSpec(short, long, defval, usage string, required bool) StringOptionSpec {
	return StringOptionSpec{
		OptionSpec{short, long, usage, true},
		new(string), defval,
	}
}
func NewStringFlag(fs *flag.FlagSet, short, long, defval, usage string, required bool) StringOptionSpec {
	f := NewStringOptionSpec(short, long, defval, usage, required)
	f.defineFlag(fs)
	return f
}

func NewInt64OptionSpec(short, long string, defval int64, usage string, required bool) Int64OptionSpec {
	return Int64OptionSpec{
		OptionSpec{short, long, usage, true},
		new(int64), defval,
	}
}
func NewInt64Flag(fs *flag.FlagSet, short, long string, defval int64, usage string, required bool) Int64OptionSpec {
	f := NewInt64OptionSpec(short, long, defval, usage, required)
	f.defineFlag(fs)
	return f
}

func NewUintOptionSpec(short, long string, defval uint, usage string, required bool) UintOptionSpec {
	return UintOptionSpec{
		OptionSpec{short, long, usage, true},
		new(uint), defval,
	}
}
func NewUintFlag(fs *flag.FlagSet, short, long string, defval uint, usage string, required bool) UintOptionSpec {
	f := NewUintOptionSpec(short, long, defval, usage, required)
	f.defineFlag(fs)
	return f
}

func NewBoolOptionSpec(short, long string, defval bool, usage string, required bool) BoolOptionSpec {
	return BoolOptionSpec{
		OptionSpec{short, long, usage, true},
		new(bool), defval,
	}
}
func NewBoolFlag(fs *flag.FlagSet, short, long string, defval bool, usage string, required bool) BoolOptionSpec {
	f := NewBoolOptionSpec(short, long, defval, usage, required)
	f.defineFlag(fs)
	return f
}

func FlagSet(cmd lsf.CommandCode) *flag.FlagSet {
	return flag.NewFlagSet(cmd.String(), flag.ContinueOnError)
}

func subCommandCode(cmd lsf.CommandCode, sub string) lsf.CommandCode {
	return lsf.CommandCode(string(cmd) + "-" + sub)
}
