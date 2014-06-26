package panics

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type StringCodec interface {
	String() string
}

type Error struct {
	Cause error
	err   error
}

func (e Error) Error() string {
	return e.err.Error()
}

func Cause(e error) error {
	ex, ok := e.(*Error)
	if !ok {
		return e
	}
	return ex.Cause
}

func OnFalse(flag bool, info ...interface{}) {
	if flag {
		return
	}
	err := fmt.Errorf("%s - assert-fail: FALSE", fmtInfo(info...))
	panic(&Error{Cause: err, err: err})
}

func OnTrue(flag bool, info ...interface{}) {
	if !flag {
		return
	}
	err := fmt.Errorf("%s - assert-fail: TRUE", fmtInfo(info...))
	panic(&Error{Cause: err, err: err})
}

func fmtInfo(info ...interface{}) string {
	var msg = ""
	if len(info) > 0 {
		for _, s := range info {
			str := ""
			switch t := s.(type) {
			case string:
				str = t
			case StringCodec:
				str = t.String()
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				str = fmt.Sprintf("%d", t)
			case time.Time:
				str = fmt.Sprintf("'%d epoch-ns'", t.UnixNano())
			case bool:
				str = fmt.Sprintf("%t", t)
			default:
				str = fmt.Sprintf("%v", t)
			}
			str = " " + str
			msg += str
		}
		msg = strings.Trim(msg, " ")
	}
	return msg
}

func OnError(e error, info ...interface{}) {
	if e == nil {
		return
	}
	var err error = e
	if len(info) > 0 {
		err = fmt.Errorf("error: %s - cause: %s", fmtInfo(info...), e)
	} else if !strings.HasPrefix(e.Error(), "error:") {
		err = fmt.Errorf("error: %s%s", fmtInfo(info...), e)
	}
	panic(&Error{Cause: e, err: err})
}

func Recover(err *error) error {
	p := recover()
	if p == nil {
		return nil
	}

	switch t := p.(type) {
	case *Error:
		//*err = Cause(t)
		*err = t
	case error:
		*err = t
	case string:
		*err = fmt.Errorf(t)
	default:
		*err = fmt.Errorf("recovered-panic: %q", t)
	}
	return *err
}

// TODO: no rush but refactor this ..
func AsyncRecover(stat chan<- interface{}, okstat interface{}) {
	p := recover()
	if p == nil {
		stat <- okstat
		return
	}

	switch t := p.(type) {
	case *Error:
		stat <- t
	case error:
		stat <- t
	case string:
		stat <- fmt.Errorf(t)
	default:
		stat <- fmt.Errorf("recovered-panic: %q", t)
	}
}

type fnpanics struct {
	fname string
}
type Panics interface {
	Recover(err *error) error
	OnError(e error, info ...interface{})
	OnFalse(flag bool, info ...interface{})
	OnTrue(flag bool, info ...interface{})
}

func (t *fnpanics) Recover(err *error) error {
	e := Recover(err)
	return e
}
func (t *fnpanics) infoFixup(info ...interface{}) []interface{} {
	infofn := []interface{}{t.fname + ":"}
	return append(infofn, info...)
}
func (t *fnpanics) OnError(e error, info ...interface{}) {
	infofn := t.infoFixup(info...)
	OnError(e, infofn...)
}
func (t *fnpanics) OnFalse(flag bool, info ...interface{}) {
	infofn := t.infoFixup(info...)
	OnFalse(flag, infofn...)
}
func (t *fnpanics) OnTrue(flag bool, info ...interface{}) {
	infofn := t.infoFixup(info...)
	OnTrue(flag, infofn...)
}

//func ForFunc(fname string) Panics {
//	return &fnpanics{fname}
//}

func ExitHandler() {
	p := recover()
	if p == nil {
		os.Exit(0)
	}

	var e error
	switch t := p.(type) {
	case *Error:
		//*err = Cause(t)
		e = t
	case error:
		e = t
	case string:
		e = fmt.Errorf(t)
	default:
		e = fmt.Errorf("recovered-panic: %q", t)
	}
	stat := 1
	log.Printf("panics.ExitHandler: exit-stat:%d cause: %s", stat, e)
	os.Exit(stat)
}
