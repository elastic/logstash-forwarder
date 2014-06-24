package test

import(
	"testing"
	"reflect"
)

// Assert the equivalence of the expected and have arguments.
// Note that testing.T.Fatal is called on assert failure.
func AssertStringResult(t *testing.T, testname, resname string, expected, have string) {
	if expected != have {
		t.Fatalf("%s:%s - expected %q have %q", testname, resname, expected, have)
	}
}

func AssertEquals(t *testing.T, testname, resname string, expected, have interface{}) {
	vexp := reflect.ValueOf(expected)
	vhave := reflect.ValueOf(have)
	kexp := vexp.Kind()
	khave:= vhave.Kind()
	if kexp != khave {
		t.Fatalf("'expected' and 'have' are not the same Kind", kexp, khave)
	}

}
