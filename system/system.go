package system

import "os"

var NilValue = []byte{}

// TODO: must be OS portable
func UserHome() string {
	return os.Getenv("HOME")
}
