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
	global  BoolOptionSpec
	id      StringOptionSpec
	freq    UintOptionSpec
	maxSize UintOptionSpec
	maxAge  UintOptionSpec
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
		global:  NewBoolFlag(Track.Flag, "G", "global", false, "command applies globally", false),
		id:      NewStringFlag(Track.Flag, "s", "stream-id", "", "unique identifier for stream", true),
		freq:    NewUintFlag(Track.Flag, "f", "frequency", 1, "report frequency - n / sec (e.g. 1000 1/ms)", true),
		maxSize: NewUintFlag(Track.Flag, "N", "max-size", 0, "max size of fs object cache", true),
		maxAge:  NewUintFlag(Track.Flag, "T", "max-age", 0, "max age of objects in fs object cache", true),
	}
}

func initTrack(env *lsf.Environment, args ...string) (err error) {
	log.Println("command/track: initTrack:")

	e := verifyRequiredOption(trackCmdOptions.id)
	panics.OnError(e, "initTrack:", "verifyRequiredOption")

	// either age or size needs to be capped.
	ageoptDefined := *trackCmdOptions.maxAge.value != trackCmdOptions.maxAge.defval
	sizeoptDefined := *trackCmdOptions.maxSize.value != trackCmdOptions.maxSize.defval
	if ageoptDefined && sizeoptDefined {
		panic("only one of age or size limits can be specified for the cache. run with -h flag for details.")
	} else if !(ageoptDefined || sizeoptDefined) {
		panic("one of age or size limits must be specified for the cache. run with -h flag for details.")
	}

	return
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

	maxSize := uint16(*trackCmdOptions.maxSize.value)
	maxAge := time.Duration(*trackCmdOptions.maxAge.value)
	var scout lsfun.TrackScout = lsfun.NewTrackScout(logStream.Path, logStream.Pattern, maxSize, fs.InfoAge(maxAge))

	freq := int(*trackCmdOptions.freq.value) // delay is time.Second
	delay := int(time.Second) / freq
	log.Printf("delay is %d", delay)

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
				time.Sleep(time.Duration(delay))
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
