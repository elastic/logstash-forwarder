package command

import (
	"log"
	"lsf"
	. "lsf/panics"
	"os"
	"os/signal"
)

const trackCmdCode lsf.CommandCode = "track"

type trackOptionSpec struct {
	verbose BoolOptionSpec
	global  BoolOptionSpec
	path    StringOptionSpec
	pattern StringOptionSpec
}

var Track *lsf.Command
var trackCmdOptions *trackOptionSpec

func init() {
	Track = &lsf.Command{
		Name:     trackCmdCode,
		About:    "track files",
		Init:     initTrack,
		Run:      runTrack,
		End:      endTrack,
		Flag:     FlagSet(trackCmdCode),
		IsActive: true,
	}
	trackCmdOptions = &trackOptionSpec{
		verbose: NewBoolFlag(Track.Flag, "v", "verbose", false, "be verbose in list", false),
		global:  NewBoolFlag(Track.Flag, "G", "global", false, "command applies globally", false),
		path:    NewStringFlag(Track.Flag, "p", "path", "", "path to log files", true),
		pattern: NewStringFlag(Track.Flag, "n", "name-pattern", "", "naming pattern of journaled log files", true),
	}
}

var user chan os.Signal

func registerSignal() {
	user = make(chan os.Signal, 1)
	signal.Notify(user, os.Interrupt, os.Kill)
}
func initTrack(env *lsf.Environment, args ...string) (err error) {

	// check opts
	log.Printf("command/track.initTrack")
	// setup the signal trap
	registerSignal()

	return
}

func runTrack(env *lsf.Environment, args ...string) (err error) {
	defer Recover(&err)
	log.Printf("command/track.initTrack")

	go func() {
		for {
			select {
			case <-user:
				log.Printf("command/track.initTrack SIG STOP")
				break
			default:
				log.Printf("command/track.initTrack RUNNING")
			}
		}
	}()

	return
}

func endTrack(env *lsf.Environment, args ...string) (err error) {
	log.Printf("command/track.endTrack END")
	return
}
