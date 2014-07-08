package lsf

import "log"

func NilInitializer() error { return nil }

// REVU: TODO move to kriterium
type Component struct {
	Initialize func() error
}

func (c *Component) debugCompConst() error {
	log.Printf("Component.debugConst: comp-type: %T", c)
	c.Initialize = NilInitializer
	return nil
}
