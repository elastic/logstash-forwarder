package anomaly

import (
	"fmt"
	"time"
)

type Error struct {
	Cause error
	err   error
}

func (e Error) Error() string {
	return e.err.Error()
}

func OnError0(e error) {
	if e == nil {
		return
	}
	panic(e)
}
func Cause(e error) error {
	ex, ok := e.(*Error)
	if !ok {
		return e
	}
	return ex.Cause
}

func PanicOnFalse(flag bool, info ...interface{}) {
	if flag {
		return
	}
	err := fmt.Errorf("error %s%s", fmtInfo(info...), "false")
	panic(&Error{Cause: err, err: err})
}

func PanicOnTrue(flag bool, info ...interface{}) {
	PanicOnFalse(!flag, info...)
}

func fmtInfo(info ...interface{}) string {
	var msg = ""
	if len(info) > 0 {
		for _, s := range info {
			str := ""
			switch t := s.(type){
			case string:
				str = t
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				str = fmt.Sprintf("%d", t)
			case time.Time:
				str = fmt.Sprintf("'%d epoch-ns'", t.UnixNano())
			case bool:
				str = fmt.Sprintf("%t", t)
			default:
				str = fmt.Sprintf("%q", t)
			}
			msg += str
		}
		msg += ": "
	}
	return msg
}

func PanicOnError(e error, info ...interface{}) {
	if e == nil {
		return
	}
	err := fmt.Errorf("%s%s", fmtInfo(info...), e)
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
