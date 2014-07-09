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

// package process supports the basic channel based
// controlled process semantics of system.Process and system.Supervisor
//
// REVU: TODO: move to kriterium
package process

import ()

// Control structure for a system "process" (goroutine)
// provides the means for a consistent light weight management of goroutines.
type Control struct {
	chansupervisor chan interface{} // Supervisor to process channel
	chanproc       chan interface{} // Process to chansupervisor channel
}

/// interface: system.Process ///////////////////////////////////

func (c *Control) Signal() chan<- interface{} {
	return c.chansupervisor
}
func (c *Control) Status() <-chan interface{} {
	return c.chanproc
}

/// interface: system.Supervisor ////////////////////////////////

func (c *Control) Command() <-chan interface{} {
	return c.chansupervisor
}
func (c *Control) Report() chan<- interface{} {
	return c.chanproc
}

func NewProcessControl() *Control {
	return &Control{
		chansupervisor: make(chan interface{}, 1),
		chanproc:       make(chan interface{}, 1),
	}
}
