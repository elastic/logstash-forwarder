package main

import (
	"flag"
	"fmt"
	"log"
	. "lsf/anomaly"
	"os"
	"path"
	"time"
)

var config struct {
	path, filename string
	maxsize        uint64
	maxfiles       uint
	delay          time.Duration
	fileperm       os.FileMode
}

// Options -name is required.
var delayOpt uint
var filemode uint

func init() {
	log.SetFlags(0)
	flag.StringVar(&config.path, "path", ".", "path to log file dir")
	flag.StringVar(&config.filename, "name", "", "basename for log files")
	flag.Uint64Var(&config.maxsize, "size", uint64(16777216), "max size of each log file")
	flag.UintVar(&config.maxfiles, "num", uint(16), "max number of rotated filesa")
	flag.UintVar(&delayOpt, "f", uint(10), "microsec delay between each log event")
	flag.UintVar(&filemode, "m", uint(0644), "microsec delay between each log event")
}

// Simulate a rotating log writer.
// See init() for option details.
func main() {

	flag.Parse()
	config.delay = time.Duration(delayOpt) * time.Microsecond
	config.fileperm = os.FileMode(0644)
	if config.filename == "" {
		log.Println("option -name is required.")
		flag.Usage()
		os.Exit(0)
	}
	stop, wdone := newProcess()
	go writeLog(config.path, config.filename, int64(config.maxsize), config.maxfiles, config.delay, stop, wdone)

	log.Printf("%s\n", config)
	var wait chan struct{}
	<-wait
}

func writeLog(dir, basename string, maxsize int64, maxfiles uint, delay_msec time.Duration, stop <-chan interface{}, wdone chan<- interface{}) {

	//	os.Open()

	fname := path.Join(dir, basename)
	file, e := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE, config.fileperm)
	if e != nil {
		file, e = os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, config.fileperm)
		PanicOnError(e, "os.OpenFile", "CREATE|TRUNC", fname)
	}

	var seq uint = 0

	// continuing from an earlier run?
	// note: won't pick up the sequence so it will overwrite ..
	//       sipmly will append to main log file
	n, _ := file.Seek(0, os.SEEK_END)
	if n > maxsize {
		file, seq, e = rotate(file, seq, config.maxfiles)
		PanicOnError(e, "rotate", "on-startup")
		n = 0
	}

	for {
		select {
		case <-stop:
			log.Printf("writer STOP n:%d\n", n)
			wdone <- true
			close(wdone)
			return
		default:
			line := simulateLogInput()

			_, e := file.Write(line)
			PanicOnError(e, "file.Write", file)

			n += int64(len(line))
			if n > maxsize {
				file, seq, e = rotate(file, seq, config.maxfiles)
				PanicOnError(e, "rotate", "on-rotate")
				n = 0
			}
			time.Sleep(delay_msec)
		}
	}
}

func newProcess() (stop <-chan interface{}, wdone chan<- interface{}) {
	return make(chan interface{}, 1), make(chan interface{}, 1)
}
func rotate(file *os.File, seq uint, seqmax uint) (newfile *os.File, newseq uint, err error) {
	Recover(&err)

	oldname := file.Name()
	if seq == seqmax-1 {
		seq = 0
	} else {
		seq++
	}
	newname := fmt.Sprintf("%s.%d", oldname, seq)
	log.Printf("writer ROTATING %s to %s\n", oldname, newname)

	var e error
	e = file.Close()
	PanicOnError(e, "os.Create", file)

	e = os.Rename(oldname, newname)
	PanicOnError(e, "os.Rename", oldname, newname)

	newfile, e = os.Create(oldname)
	PanicOnError(e, "os.Create", oldname)

	return newfile, seq, nil
}

var sequence uint64

func simulateLogInput() []byte {
	line := fmt.Sprintf("%d %019d INFO simulated single line sequenced log entry\n", time.Now().UnixNano(), sequence)
	sequence++
	return []byte(line)
}
