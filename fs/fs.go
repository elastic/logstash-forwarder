package fs

import (
	"log"
	"lsf/panics"
	"os"
	"path"
	"path/filepath"
)

// Returns []string{} if pattern is ""/zv.
// REVU:
// if sampling period is (e.g. 1/microsec) extremely small
// then os.Stat may hit a phantom file: it was deleted right
// after filepath.Glob returned. There is nothing that can
// be done here but to ignore it.
func FindMatchingPaths(basepath, pattern string) (matches []string, err error) {
	defer panics.Recover(&err)

	if pattern == "" {
		return []string{}, nil
	}

	glob := path.Join(basepath, pattern)
	log.Printf("track: %q\n", glob)
	fspaths0, e := filepath.Glob(glob)
	panics.OnError(e, "FindMatchingPaths", "path.Join", glob)

	fspaths := make([]string, len(fspaths0))
	n := 0
	for _, fspath := range fspaths0 {
		_, e := os.Stat(fspath)
		switch {
		case e == nil:
			fspaths[n] = fspath
			n++
		case os.IsNotExist(e): // ignore - see REVU note
		default: // what is it?
			panics.OnError(e, "FindMatchingPaths", "os.Stat", fspath)
		}
	}
	return fspaths[:n], nil
}
