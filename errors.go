package lsf

import (
	"github.com/elasticsearch/kriterium/errors"
)

// ----------------------------------------------------------------------------
// error codes
// ----------------------------------------------------------------------------

var ERR = struct {
	USAGE,
	INVALID,
	RELATIVE_PATH,
	EXISTING_LSF,
	NOT_EXISTING_LSF,
	EXISTING,
	NOT_EXISTING,
	ILLEGAL_STATE,
	ILLEGAL_STATE_REGISTRAR_RUNNING,
	EXISTING_STREAM,
	CONCURRENT errors.TypedError
}{
	USAGE:                           errors.New("invalid command usage"),
	INVALID:                         errors.New("invalid argument"),
	RELATIVE_PATH:                   errors.New("path is not absolute"),
	EXISTING_LSF:                    errors.New("lsf environment already exists"),
	NOT_EXISTING_LSF:                errors.New("lsf environment does not exists at location"),
	EXISTING:                        errors.New("lsf resource already exists"),
	NOT_EXISTING:                    errors.New("lsf resource does not exist"),
	ILLEGAL_STATE:                   errors.New("illegal state"),
	ILLEGAL_STATE_REGISTRAR_RUNNING: errors.New("Registrar already running"),
	EXISTING_STREAM:                 errors.New("stream already exists"),
	CONCURRENT:                      errors.New("concurrent operation error"),
}
