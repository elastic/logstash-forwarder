// package process supports the basic channel based
// controlled process semantics of system.Process and system.Supervisor
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
