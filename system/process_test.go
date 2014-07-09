// Licensed to Elasticsearch under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package system

import (
	"lsf/system/process"
	"lsf/test"
	"testing"
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
