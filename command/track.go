package command

import (
	"log"
	"lsf"
	"lsf/fs"
	"lsf/lsfun"
	"lsf/panics"
	"time"
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
	// TODO: just get the stream id. optional flag for persistence of state and events
	trackCmdOptions = &trackOptionSpec{
		verbose: NewBoolFlag(Track.Flag, "v", "verbose", false, "be verbose in list", false),
		global:  NewBoolFlag(Track.Flag, "G", "global", false, "command applies globally", false),
		path:    NewStringFlag(Track.Flag, "p", "path", "", "path to log files", true),
		pattern: NewStringFlag(Track.Flag, "n", "name-pattern", "", "naming pattern of journaled log files", true),
	}
}

func initTrack(env *lsf.Environment, args ...string) (err error) {
	log.Println("command/track: initTrack:")
	// TODO:
	// 1 - verify ~/.lsf (LS/F environment)
	// 2- get stream info from ~/.lsf

	return nil
}

// Track runs continuously, generating a tracking scout report per
// configuration.
var opt = struct {
	basepath  string
	pattern   string
	maxSize   uint
	maxAge    fs.InfoAge
	delaymsec uint
	about     func() string
}{
	basepath:  "/Users/alphazero/Code/es/go/src/lsf/development/tools/rotating-logger",
	pattern:   "apache2*",
	maxSize:   17,
	maxAge:    fs.InfoAge(0),
	delaymsec: 100,
}

func runTrack(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)
	log.Printf("command/track.runTrack")

	supervisor := getSupervisor(env) // panics // REVU: generic to all active cmds
	// TODO: all args must be from env or options
	var scout lsfun.TrackScout = lsfun.NewTrackScout(opt.basepath, opt.pattern, uint16(opt.maxSize), opt.maxAge)

	everUntilInterrupted := true
	go func() {
		for everUntilInterrupted {
			select {
			case <-supervisor.Command():
				everUntilInterrupted = false
			default:
				report, e := scout.Report()
				panics.OnError(e, "main", "scout.Report")

				for _, event := range report.Events {
					if event.Code != lsfun.TrackEvent.KnownFile { // printing NOP events gets noisy
						log.Println(event)
					}
				}
				time.Sleep(time.Millisecond * time.Duration(opt.delaymsec))
			}
		}
		supervisor.Report() <- "done"
	}()

	return
}

func endTrack(env *lsf.Environment, args ...string) (err error) {
	log.Printf("command/track.endTrack END")
	return
}
