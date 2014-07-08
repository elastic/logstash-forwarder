package lslib

import (
	"fmt"
	"github.com/elasticsearch/kriterium/panics"
	"io"
	"os"
	"os/signal"
	"path"
)

// ----------------------------------------------------------------------
// API
// ----------------------------------------------------------------------

// An append log writer with auto-rotation
// Exposes the io.Writer interface.
type RotatingFileWriter interface {
	io.Writer
	// closes the Writer.
	// If already closed, it is a nop.
	// Calls to Write() after close() will panic.
	Close()
	// Calls Close() if notified of any of the
	// following os.Signals. Should be invoked only once.
	CloseOnSignal(signals ...os.Signal)
}

// ----------------------------------------------------------------------
// Support
// ----------------------------------------------------------------------
type rotatingFileWriter struct {
	basepath, filepath string
	file               *os.File
	sequence, limit    uint
	maxFileSize        int64
	offset             int64
	closed             bool
}

// Returns a new RotatingFileWiter with default settings:
// max number of files: 16
// max size: 16777216 bytes
// See NewRotatingFileWriter for additional details
func GetFileRotatorWithDefaults(basepath, basename string) (r RotatingFileWriter, err error) {
	return NewRotatingFileWriter(basepath, basename, uint(16), int64(1<<24))
}

// Returns a new RotatingFileWiter.
// Files are closed on rotation. Active file will close on os.Interrupt | os.Kill.
// This implementation of RotatingFileWriter does not support concurrent client.
func NewRotatingFileWriter(basepath, basename string, maxseq uint, maxFileSize int64) (rotator *rotatingFileWriter, err error) {

	defer panics.Recover(&err)

	fileperm := os.FileMode(0644)
	fname := path.Join(basepath, basename)

	file, e := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE, fileperm)
	if e != nil {
		file, e = os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileperm)
		panics.OnError(e, "os.OpenFile", "CREATE|TRUNC", fname)
	}

	//	filepath := path.Join(basepath, file.Name())
	offset, _ := file.Seek(0, os.SEEK_END)
	//	rotator = &rotatingFileWriter{basepath, filepath, file, 0, maxseq, maxFileSize, offset, false}
	rotator = &rotatingFileWriter{basepath, basename, file, 0, maxseq, maxFileSize, offset, false}

	e = rotator.rotateOnLimit()
	panics.OnError(e, "NewFileRotator", "rotateOnLimit:")

	return rotator, nil
}

// ----------------------------------------------------------------------
// interface: RotatingFileWriter
// ----------------------------------------------------------------------
func (r *rotatingFileWriter) Write(p []byte) (n int, err error) {
	defer panics.Recover(&err)

	if r.closed {
		err = fmt.Errorf("rotatingFIleWriter: Write: illegal state: closed")
		return
	}
	n, err = r.file.Write(p)
	if err != nil {
		return
	}
	r.offset += int64(n)
	r.rotateOnLimit()
	return
}
func (r *rotatingFileWriter) Close() {
	if r.closed {
		return
	}
	r.file.Close()
	r.closed = true
}

func (r *rotatingFileWriter) CloseOnSignal(signals ...os.Signal) {
	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, signals...)
	go func() {
		select {
		case <-osSignal:
			r.Close()
			cleanClose(osSignal)
			return
		}
	}()
}
func cleanClose(ch chan os.Signal) {
	defer func() { recover() }()
	close(ch)
}

// ----------------------------------------------------------------------
// inner ops
// ----------------------------------------------------------------------
func (r *rotatingFileWriter) rotateOnLimit() error {
	if r.offset > r.maxFileSize {
		fmt.Printf("DEBUG: rotatingFileWriter: rotateOnLimit: file:%s offset:%d limit:%d\n", r.file.Name(), r.offset, r.maxFileSize)
		file, e := r.rotate()
		if e != nil {
			return e
		}
		r.file = file
		r.offset = 0
	}
	return nil
}

func (r *rotatingFileWriter) rotate() (newfile *os.File, err error) {

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
