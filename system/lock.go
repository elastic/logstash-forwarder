package system

import (
	"fmt"
	"os"
	"time"
)

// structure of the FS based lock
type lock struct {
	resource string
	accessed time.Time // recorded - not used
	deadline time.Time // recorded - not used
	data     []byte
	fileinfo *os.FileInfo
}

type Lock interface {
	Unlock() error
}

var E_WAIT_EXPIRE = fmt.Errorf("wait deadline expired")
var E_INVALID_LOCK_STATE = fmt.Errorf("lock state not as expected")

func (l *lock) Unlock() error {
	if l == nil {
		panic("BUG lock.Unlock: nil receiver")
	}
	if l.fileinfo == nil {
		return E_INVALID_LOCK_STATE
	}
	if l.resource == "" {
		panic("BUG lock.Unlock: nil resource")
	}

	e := removeLockFile(l.resource)
	if e != nil {
		return fmt.Errorf("error lock.Unlock: removeLockFile: %q - %s", l.resource, e.Error())
	}

	l.fileinfo = nil
	l.data = nil
	l.accessed = time.Now()

	return nil
}

// attempts to create a lock file for the named resource.
// returns:
// lock, true, nil on success
// nil, false, nil on resource already locked
// nil, false, error on error
func LockResource(resource string, info string) (l Lock, ok bool, err error) {

	//	log.Printf("DEBUG LockResource: %s", resource)
	// try creating lock file
	// REVU: this can be reenterant TODO check the pid in lock file.
	file, e := createLockFile(resource)
	if os.IsExist(e) {
		return nil, false, nil
	} else if e != nil {
		return nil, false, e
	}
	//	log.Printf("DEBUG LockResource: %", file.Name())

	fileinfo, e := file.Stat()
	if e != nil {
		return nil, false, e // BUG
	}

	now := time.Now()
	data := []byte(fmt.Sprintf("%d %d %d %q\n", now.UnixNano(), now.UnixNano(), os.Getpid(), info))

	n, e := file.Write(data)
	//	file.Sync()
	if e != nil {
		return nil, false, e
	}
	if e == nil && n < len(data) {
		return nil, false, fmt.Errorf("error - system.writeFile: short write %d of %d", n, len(data))
	}
	if e != nil {
		return nil, false, fmt.Errorf("error system.LockResource: writeLockInfo: %q - %s", resource, e.Error())
	}

	defer file.Close()

	return &lock{resource, now, now, data, &fileinfo}, true, nil
}

func removeLockFile(resource string) error {
	//	log.Printf("DEBUG removeLockFile: %s", resource)
	filename := resource + ".lock"
	//	log.Printf("DEBUG removeLockFile: %s", filename)
	if _, e := os.Stat(filename); os.IsNotExist(e) {
		return e
	}
	if e := os.Chmod(filename, os.FileMode(0644)); e != nil {
		return e
	}
	if e := os.Remove(filename); e != nil {
		return e
	}
	return nil
}

func createLockFile(resource string) (*os.File, error) {
	//	log.Printf("DEBUG createLockFile: %s", resource)
	filename := resource + ".lock"
	//	log.Printf("DEBUG createLockFile: %s", filename)
	file, e := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(0644))
	return file, e
}
