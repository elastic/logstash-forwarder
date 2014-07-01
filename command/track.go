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
	trackCmdOptions = &trackOptionSpec{
		global:  NewBoolFlag(Track.Flag, "G", "global", false, "command applies globally", false),
		id:      NewStringFlag(Track.Flag, "s", "stream-id", "", "unique identifier for stream", true),
		freq:    NewUintFlag(Track.Flag, "f", "frequency", 1, "report frequency - n / sec (e.g. 1000 1/ms)", true),
		maxSize: NewUintFlag(Track.Flag, "N", "max-size", 0, "max size of fs object cache", true),
		maxAge:  NewUintFlag(Track.Flag, "T", "max-age", 0, "max age of objects in fs object cache", true),
	}
}

func initTrack(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)

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

	id := *trackCmdOptions.id.value

	// Load stream doc and get LogStream instance
	docId := docIdForStream(id)
	doc, e := env.LoadDocument(docId)
	panics.OnError(e, "no such stream:", id)
	panics.OnNil(doc, "BUG - doc is nil")

	logStream := schema.DecodeLogStream(doc)
	_, e = env.Set(lsf.VarKey(docId), logStream)
	panics.OnError(e, "env.Set(docId)")

	// Run in exclusive mode - lock stream's op
	opLock, opLockId, e := env.ExclusiveResourceOp(system.Op.StreamTrack, id, "track command")
	if e != nil {
		return e
	}

	_, e = env.Set(lsf.VarKey(opLockId), opLock)
	panics.OnError(e, "env.Set(lockid)")

	return
}

func runTrack(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)
	log.Printf("command/track.runTrack")

	id := *trackCmdOptions.id.value
	supervisor := getSupervisor(env) // panics // REVU: generic to all active cmds

	docId := docIdForStream(id)
	v, found := env.Get(lsf.VarKey(docId))
	panics.OnFalse(found, "BUG", "logStream not bound", docId)
	logStream := v.(*schema.LogStream)

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
				panics.OnError(e, "main", "scout.Report") // REVU: wrong. send error via channel and close

				log.Println("--- events -------------------------------------------")
				for _, event := range report.Events {
					if event.Code != lsfun.TrackEvent.KnownFile { // printing NOP events gets noisy
						log.Println(event)
					}
				}

				objects := scout.ObjectMap()

				log.Println("--- objects ------------------------------------------")
				for _, fsobj := range objects {
					log.Println(fsobj.String())
				}
				log.Println()
				time.Sleep(time.Duration(delay))
			}
		}
		supervisor.Report() <- lsf.Status.Interrupted_Ok
	}()

	return
}

// cleanup:
func endTrack(env *lsf.Environment, args ...string) (err error) {
	defer panics.Recover(&err)

	id := *trackCmdOptions.id.value

	// - unlock track action for stream
	lockid := trackResourceId(env, id, "track")
	v, found := env.Get(lsf.VarKey(lockid))
	panics.OnFalse(found, "BUG", "lock not bound", lockid)

	return v.(system.Lock).Unlock()
}

func docIdForStream(id string) string {
	return fmt.Sprintf("stream.%s.stream", id)
}
func trackResourceId(env *lsf.Environment, streamId string, restype string) string {
	resource := fmt.Sprintf("stream.%s.%s", streamId, restype)
	return env.ResourceId(resource)
}
