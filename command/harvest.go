package command

import (
	"lsf"
)

const cmd_harvest lsf.CommandCode = "harvest"

type harvestOptionsSpec struct {
	path   StringOptionSpec
	stream StringOptionSpec
}

var Harvest *lsf.Command
var harvestOptions *harvestOptionsSpec

func init() {

	Harvest = &lsf.Command{
		Name: cmd_harvest,
		Run:  runHarvest,
		Flag: FlagSet(cmd_harvest),
	}

	harvestOptions = &harvestOptionsSpec{
		path:   NewStringOptionSpec("p", "path", ".", "path to log-stream files", false),
		stream: NewStringOptionSpec("s", "stream", "", "the log-stream identifier", false),
	}
	harvestOptions.path.defineFlag(Harvest.Flag)
	harvestOptions.stream.defineFlag(Harvest.Flag)
}

func runHarvest(env *lsf.Environment, args ...string) error {
	/*
		prospecter.GoHarvest(in, out, err, stream, path,
	*/
	//	env.Vars["some.Key()"]

	panic("command.harvest() not impelemented!")

}
