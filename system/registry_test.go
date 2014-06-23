package system

import (
	"path"
	"testing"
)

// REVU: this belongs to system-test.go
func TestObjectPathForKey(t *testing.T) {
	lsfpath := "/Users/alphazero/.lsf"
	// just test the 2 possible patterns:
	// 1 - top level resources
	// 2 - dot notation res ids
	var ids = []string{
		"system",
		"remote.remote-abc.remote",
		"stream.my-server.stream",
		"foo.bar.paz",
	}

	var expected = []struct {
		basepath, basename string
	}{
		{path.Join(lsfpath, "/"), "SYSTEM"},
		{path.Join(lsfpath, "remote/remote-abc"), "REMOTE"},
		{path.Join(lsfpath, "stream/my-server"), "STREAM"},
		{path.Join(lsfpath, "foo/bar"), "PAZ"},
	}

	for n, id := range ids {
		basepath, basename := LogPathForKey(lsfpath, LogId(id))
		assertStringResult(t, "TestLogPathForKey", "basepath", expected[n].basepath, basepath)
		assertStringResult(t, "TestLogPathForKey", "basename", expected[n].basename, basename)
	}

}

// TODO: move to a test util
func assertStringResult(t *testing.T, testname, resname string, expected, have string) {
	if expected != have {
		t.Fatalf("%s:%s - expected %q have %q", testname, resname, expected, have)
	}
}
