package command

import (
	"fmt"
	"log"
	"lsf"
	"lsf/fs"
	"lsf/lsfun"
	"lsf/panics"
	"lsf/schema"
	"lsf/system"
	"time"
)

const trackCmdCode lsf.CommandCode = "track"

type trackOptionSpec struct {
	global BoolOptionSpec
	id     StringOptionSpec
	//	frequency Int64OptionSpec
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
		global: NewBoolFlag(Track.Flag, "G", "global", false, "command applies globally", false),
		id:     NewStringFlag(Track.Flag, "s", "stream-id", "", "unique identifier for stream", true),
	}
}

func initTrack(env *lsf.Environment, args ...string) (err error) {
	log.Println("command/track: initTrack:")

	e := verifyRequiredOption(trackCmdOptions.id)
	panics.OnError(e, "initTrack:", "verifyRequiredOption")

	return
}

// Track runs continuously, generating a tracking scout report per
// configuration.
var opt = struct {
	//	basepath  string
	//	pattern   string
	maxSize   uint
	maxAge    fs.InfoAge
	delaymsec uint
	about     func() string
}{
	//	basepath:  "/Users/alphazero/Code/es/go/src/lsf/development/tools/rotating-logger",
	//	pattern:   "apache2*",
	maxSize:   17,
	maxAge:    fs.InfoAge(0),
	delaymsec: 100,
}

func runTrack(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)
	log.Printf("command/track.runTrack")

	supervisor := getSupervisor(env) // panics // REVU: generic to all active cmds

	// Load stream doc and get LogStream instance
	id := schema.StreamId(*trackCmdOptions.id.value)
	docid := system.DocId(fmt.Sprintf("stream.%s.stream", id))
	doc, e := env.LoadDocument(docid)
	panics.OnError(e, "BUG command.initTrack:", "LoadDocument:", string(docid))
	panics.OnTrue(doc == nil, "BUG command.initTrack:", "LoadDocument:", string(docid))

	logStream := schema.DecodeLogStream(doc)
	log.Println(logStream.String())

	// Run in exclusive mode
	resource := fmt.Sprintf("stream.%s.track", id)
	lockid := env.ResourceId(resource)
	oplock, ok, e := system.LockResource(lockid, "track stream - resource "+resource)
	panics.OnError(e, "command.runUpdateStream:", "lockResource:", resource)
	panics.OnFalse(ok, "command.runUpdateStream:", "lockResource:", resource)
	defer oplock.Unlock()

	var scout lsfun.TrackScout = lsfun.NewTrackScout(logStream.Path, logStream.Pattern, uint16(opt.maxSize), opt.maxAge)

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
