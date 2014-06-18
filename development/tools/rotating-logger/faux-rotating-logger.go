package main

import (
	"flag"
	"fmt"
	"log"
	"lsf/panics"
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

	log.Println(<-wdone)
}

func writeLog(dir, basename string, maxsize int64, maxfiles uint, delay_msec time.Duration, stop <-chan interface{}, wdone chan<- interface{}) {
	defer panics.AsyncRecover(wdone, "ok")

	fname := path.Join(dir, basename)
	file, e := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE, config.fileperm)
	if e != nil {
		file, e = os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, config.fileperm)
		panics.OnError(e, "os.OpenFile", "CREATE|TRUNC", fname)
	}

	rotator, e := GetFileRotator(file, dir, 0, config.maxfiles)
	panics.OnError(e, "GetFileRotator")

	// continuing from an earlier run?
	// note: won't pick up the sequence so it will overwrite ..
	//       sipmly will append to main log file
	n, _ := file.Seek(0, os.SEEK_END)
	if n > maxsize {
		file, e = rotator.Next()
		panics.OnError(e, "rotate", "on-startup")
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
			panics.OnError(e, "file.Write", file)

			n += int64(len(line))
			if n > maxsize {
				file, e = rotator.Next()
				panics.OnError(e, "rotate", "on-rotate")
				n = 0
			}
			time.Sleep(delay_msec)
		}
	}
}

func newProcess() (stop <-chan interface{}, wdone chan interface{}) {
	return make(chan interface{}, 1), make(chan interface{}, 1)
}

// ----------------------------------------------------------------------
// File Rotator
// ----------------------------------------------------------------------

type fileRotator struct {
	basepath, filepath string
	file               *os.File
	sequence, limit    uint
}

type FileRotator interface {
	Next() (*os.File, error)
}

func GetFileRotatorWithDefaults(file *os.File) (r FileRotator, err error) {
	return GetFileRotator(file, ".", uint(0), uint(16))
}

func GetFileRotator(file *os.File, basepath string, initseq, maxseq uint) (r FileRotator, err error) {
	defer panics.Recover(&err)

	// assert a few facts
	panics.OnTrue(file == nil, "GetFileRotator", "file is nil")
	panics.OnTrue(basepath == "", "GetFileRotator", "basepath is nil")
	panics.OnTrue(maxseq < initseq, "GetFileRotator", "maxseq < initseq")
	_, e := os.Open(basepath) // check basepath exists
	panics.OnError(e, "GetFileRotator")
	info, e := file.Stat()
	panics.OnError(e, "GetFileRotator")
	filepath := path.Join(basepath, file.Name())
	xinfo, e := os.Stat(filepath)
	panics.OnError(e, "GetFileRotator")
	os.SameFile(info, xinfo)

	return &fileRotator{basepath, filepath, file, initseq, maxseq}, nil
}
func (r *fileRotator) Next() (newfile *os.File, err error) {

	newfile, newseq, e := rotate(r.file, r.sequence, r.limit)
	if err != nil {
		return nil, fmt.Errorf("rotating file %s - cause: %s", r.filepath, e.Error())
	}
	r.file = newfile
	r.sequence = newseq

	return
}

func rotate(file *os.File, seq uint, seqmax uint) (newfile *os.File, newseq uint, err error) {
	defer panics.Recover(&err)

	oldname := file.Name()
	if seq == seqmax-1 {
		seq = 0
	} else {
		seq++
	}
	newname := fmt.Sprintf("%s.%d", oldname, seq)
	log.Printf("writer ROTATING %s to %s\n", oldname, newname)

	var e error
	info, e := file.Stat()
	panics.OnError(e, "file.Stat", file)
	filemode := info.Mode()
	e = file.Close()
	panics.OnError(e, "file.Close", file)

	e = os.Rename(oldname, newname)
	panics.OnError(e, "os.Rename", oldname, newname)

	newfile, e = os.OpenFile(oldname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filemode)
	panics.OnError(e, "os.OpenFile(O_CREATE..)", oldname)

	return newfile, seq, nil
}

var sequence uint64

func simulateLogInput() []byte {
	line := fmt.Sprintf("%d %019d INFO simulated single line sequenced log entry\n", time.Now().UnixNano(), sequence)
	sequence++
	return []byte(line)
}
