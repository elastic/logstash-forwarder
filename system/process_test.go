package system

import (
	"testing"
	"lsf/system/process"
	"lsf/test"
)

/* blackbox test of system.Process/system.Supervisor impls. */

func TestProviderSystemProcessControl(t *testing.T) {
	assert := test.GetAssertionFor(t, "TestProviderSystemProcessControl")

	// Get provider instance and cast to facet
	procctl := process.NewProcessControl()
	process := Process(procctl)
	supervisor := Supervisor(procctl)

	// process input is supervisor output
	assert.SameReference("control block command channel", process.Signal(), supervisor.Command())

	// supervisor input is process output
	assert.SameReference("control block command channel", process.Status(), supervisor.Report())
}
