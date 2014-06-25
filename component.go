package lsf

import "log"

func NilInitializer() error { return nil }

type Component struct {
	Initialize func() error
}

func (c *Component) debugCompConst() error {
	log.Printf("Component.debugConst: comp-type: %T", c)
	c.Initialize = NilInitializer
	return nil
}
