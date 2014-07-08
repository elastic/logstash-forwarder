package command

import (
	"log"
	"github.com/elasticsearch/kriterium/panics"
	"lsf"
)

const cmd_init lsf.CommandCode = "init"

type initOptionsSpec struct {
	home  StringOptionSpec
	force BoolOptionSpec
}

var Init *lsf.Command
var initOptions *initOptionsSpec

func init() {

	Init = &lsf.Command{
		Name:        cmd_init,
		About:       "Creates and initializes a LS/F port",
		Init:        nil_CommandInitFn,
		Run:         runInit,
		Flag:        FlagSet(cmd_init),
		Initializer: true,
	}
	initOptions = &initOptionsSpec{
		home:  NewStringFlag(Init.Flag, "h", "home", ".", "path directory to create lsf environemnt", false),
		force: NewBoolFlag(Init.Flag, "f", "force", false, "force the operation of command", false),
	}
}

// create and initialize an LSF base.
// The base will be created in the 'path' option location.
// 'force' flag must be set.
// Init in existing directory will raise error E_EXISTING
func runInit(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)

	home := lsf.AbsolutePath(*initOptions.home.value)
	force := *initOptions.force.value

	what := "Initialized"

	// init w/ existing is an error unless -force flag is set
	if env.Exists(home) {
		panics.OnFalse(force, "init.runInit:", "existing environment. use -force flag to reinitialize")
		what = "Re-Initialize"
	}

	envpath, e := lsf.CreateEnvironment(home, force)
	panics.OnError(e, "command/init.runInit", "on lsf.CreateEnvironment")

	log.Printf("%s LSF environment at %s\n", what, envpath)

	return
}
