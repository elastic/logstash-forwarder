package main

import (
	"flag"
	"fmt"
	"log"
	"lsf/lslib"
	"lsf/panics"
	"os"
	"time"
)

var sequence int64

func simulateLogInput() []byte {
	line := fmt.Sprintf("%d %019d INFO simulated single line sequenced log entry\n", time.Now().UnixNano(), sequence)
	sequence++
	return []byte(line)
}

var config struct {
	path, filename string
	maxsize        int64
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
	flag.Int64Var(&config.maxsize, "size", int64(16777216), "max size of each log file")
	flag.UintVar(&config.maxfiles, "num", uint(16), "max number of rotated filesa")
	flag.UintVar(&delayOpt, "f", uint(10), "microsec delay between each log event")
	flag.UintVar(&filemode, "m", uint(0644), "microsec delay between each log event")

	panics.DEBUG = true
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
	go writeLog(config.path, config.filename, config.maxsize, config.maxfiles, config.delay, stop, wdone)

	log.Println(<-wdone)
}

func writeLog(basepath, basename string, maxFileSize int64, maxfiles uint, delay_msec time.Duration, stop <-chan interface{}, wdone chan<- interface{}) {
	defer panics.AsyncRecover(wdone, "ok")

	rotator, e := lslib.NewRotatingFileWriter(basepath, basename, maxfiles, maxFileSize)
	panics.OnError(e, "NewFileRotator")

	for {
		select {
		case <-stop:
			log.Printf("writer STOP\n")
			wdone <- true
			close(wdone)
			return
		default:
			_, e := rotator.Write(simulateLogInput())
			panics.OnError(e, "rotator.Write")

			time.Sleep(delay_msec)
		}
	}
}

func newProcess() (stop <-chan interface{}, wdone chan interface{}) {
	return make(chan interface{}, 1), make(chan interface{}, 1)
}

//// ----------------------------------------------------------------------
//// Rotating File Writer
//// ----------------------------------------------------------------------
//
//type rotatingFileWriter struct {
//	basepath, filepath string
//	file               *os.File
//	sequence, limit    uint
//	maxFileSize        int64
//	offset             int64
//}
//
//type RotatingFileWriter interface {
//	io.Writer
////	rotate() (*os.File, error)
//}
//
//func GetFileRotatorWithDefaults(basepath, basename string) (r RotatingFileWriter, err error) {
//	return NewRotatingFileWriter(basepath, basename, uint(16), int64(1<<24))
//}
//
//func NewRotatingFileWriter(basepath, basename string, maxseq uint, maxFileSize int64) (rotator *rotatingFileWriter, err error) {
//
//	defer panics.Recover(&err)
//
//	fname := path.Join(basepath, basename)
//	file, e := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE, config.fileperm)
//	if e != nil {
//		file, e = os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, config.fileperm)
//		panics.OnError(e, "os.OpenFile", "CREATE|TRUNC", fname)
//	}
//
//	_, e = os.Open(basepath) // check basepath exists
//	panics.OnError(e, "NewFileRotator")
//
//	info, e := file.Stat()
//	panics.OnError(e, "NewFileRotator")
//
//	filepath := path.Join(basepath, file.Name())
//	xinfo, e := os.Stat(filepath)
//	panics.OnError(e, "NewFileRotator")
//	os.SameFile(info, xinfo)
//
//
//	offset, _ := file.Seek(0, os.SEEK_END)
//	rotator = &rotatingFileWriter{basepath, filepath, file, 0, maxseq, maxFileSize, offset}
//
//	e = rotator.rotateOnLimit()
//	panics.OnError(e, "NewFileRotator", "rotateOnLimit:")
//	return rotator, nil
//}
//
//func (r *rotatingFileWriter) Write(p []byte) (n int, err error) {
//	defer panics.Recover(&err)
//
//	n, err = r.file.Write(p)
//	if err != nil {
//		return
//	}
//	r.offset += int64(n)
//	r.rotateOnLimit()
//	return
//}
//
//func (r *rotatingFileWriter) rotateOnLimit() error {
//	if r.offset > r.maxFileSize {
//		fmt.Printf("DEBUG: rotatingFileWriter: rotateOnLimit: file:%s offset:%d limit:%d\n", r.file.Name(), r.offset, r.maxFileSize)
//		file, e := r.rotate()
//		if e != nil {
//			return e
//		}
//		r.file = file
//		r.offset = 0
//	}
//	return nil
//}
//
//func (r *rotatingFileWriter) rotate() (newfile *os.File, err error) {
//
//	newfile, newseq, e := rotate(r.file, r.sequence, r.limit)
//	if err != nil {
//		return nil, fmt.Errorf("rotating file %s - cause: %s", r.filepath, e.Error())
//	}
//	r.file = newfile
//	r.sequence = newseq
//
//	return
//}
//
//func rotate(file *os.File, seq uint, seqmax uint) (newfile *os.File, newseq uint, err error) {
//	defer panics.Recover(&err)
//
//	oldname := file.Name()
//	if seq == seqmax-1 {
//		seq = 0
//	} else {
//		seq++
//	}
//	newname := fmt.Sprintf("%s.%d", oldname, seq)
//
//	var e error
//	info, e := file.Stat()
//	panics.OnError(e, "file.Stat", file)
//	filemode := info.Mode()
//	e = file.Close()
//	panics.OnError(e, "file.Close", file)
//
//	e = os.Rename(oldname, newname)
//	panics.OnError(e, "os.Rename", oldname, newname)
//
//	newfile, e = os.OpenFile(oldname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filemode)
//	panics.OnError(e, "os.OpenFile(O_CREATE..)", oldname)
//
//	return newfile, seq, nil
//}
