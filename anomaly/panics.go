package anomaly

import (
	"fmt"
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
func PanicOnFalse(flag bool, info ...string) {
	if flag {
		return
	}
	err := fmt.Errorf("error %s%s", fmtInfo(info...), "false")
	panic(&Error{Cause: err, err: err})
}

func fmtInfo(info ...string) string {
	var msg = ""
	if len(info) > 0 {
		var m []byte
		for _, s := range info {
			m = append(m, s...)
			m = append(m, ' ')
		}
		msg = string(m[:len(m)-1]) + ": "
	}
	return msg
}

func PanicOnError(e error, info ...string) {
	if e == nil {
		return
	}
	err := fmt.Errorf("error %s%s", fmtInfo(info...), e)
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
