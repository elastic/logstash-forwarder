package system

import "os"

// TODO: must be OS portable
func UserHome() string {
	return os.Getenv("HOME")
}
