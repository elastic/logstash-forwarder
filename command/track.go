package command

import (
//	"os"
	"log"
	"lsf"
	"lsf/panics"
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

/* Ref: prototype options
	flag.StringVar(&options.basepath, "p", options.basepath, "base path to track")
	flag.StringVar(&options.pattern, "n", options.pattern, "filename glob pattern")
	flag.UintVar(&options.delaymsec, "delay", options.delaymsec, "delay in msecs between reports")
	flag.UintVar(&options.maxSize, "max-size", options.maxSize, "maximum number of fs.Objects in cache")
	flag.Var(&options.maxAge, "max-age", "limit on age of object in cache")

 */
func init() {
	Track = &lsf.Command{
		Name:     trackCmdCode,
		About:    "track files",
		Init:     initActiveCommandFn(initTrack),
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

// TODO move to command
//var sigch chan os.Signal
func initActiveCommand(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)

	log.Printf("command/track.initTrack")
	sigch, found := env.Get(lsf.VarUserSigChan)
	panics.OnFalse(found, "BUG", "env.Get(lsf.VarUserSigChan)")
	panics.OnFalse(sigch != nil, "BUG", "env.Get(lsf.VarUserSigChan)")

	return
}

func initActiveCommandFn(cmdInitFn lsf.CommandInitFn) lsf.CommandInitFn {
	return func(env *lsf.Environment, args ...string) (err error) {
		defer panics.Recover(&err)
		initActiveCommand(env, args...)
		e := cmdInitFn(env, args...)
		return e
	}
}

func runActiveCommandFn(cmdRunFn lsf.CommandFn) lsf.CommandFn {

	// REVU: this is wrong ..
	return func(env *lsf.Environment, args ...string) (err error) {
		return cmdRunFn(env, args...)
	}
}


func initTrack(env *lsf.Environment, args ...string) (err error) {
	log.Println("command/track: initTrack:")
	// TODO: put this scout into
//	opt := options //
//	var scout lsfun.TrackScout = lsfun.NewTrackScout(opt.basepath, opt.pattern, uint16(opt.maxSize), opt.maxAge)
	return nil
}
//
func runTrack(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)
	log.Printf("command/track.runTrack")

	go func() {
		log.Printf("command/track.runTrack:::go func()")
//		for {
//			select {
//			case <-user:
//				log.Printf("command/track.initTrack SIG STOP")
//				break
//			default:
//				log.Printf("command/track.initTrack RUNNING")
//			}
//		}
	}()

	return
}

func endTrack(env *lsf.Environment, args ...string) (err error) {
	log.Printf("command/track.endTrack END")
	return
}
