package process

import (
	"lsf/test"
	"testing"
)

// this is the only meaningful whitebox test we can do.
func TestConstruct(t *testing.T) {
	assert := test.GetAssertionFor(t, "TestConstruct")

	// never return nil
	control := NewProcessControl()
	assert.NotNil("control", control)

	// supervisor input is process output

}
