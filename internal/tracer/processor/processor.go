// Package processor stores processing that is required before and after
// a compiled function is called.
package processor

import (
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

type (

	// Processor stores initializers and traces to process before and
	// after calling a compiled function.
	Processor struct {
		traces
		fn    ir.Func
		inits []Initializer
		args  []Argument
	}

	// Initializer is called at the beginning of a run before
	// arguments for the backend are computed.
	Initializer interface {
		Init(*elements.InputValues) error
	}

	// Argument provides an argument to pass to the backend.
	Argument interface {
		Shape() *shape.Shape

		ToDeviceHandle(platform.Device, *elements.InputValues) (platform.DeviceHandle, error)
	}
)

// RegisterInit registers an Initializer to the graph.
func (p *Processor) RegisterInit(init Initializer) {
	p.inits = append(p.inits, init)
}

// RegisterArg an argument for the backend.
// Returns the index of the argument.
func (p *Processor) RegisterArg(arg Argument) int {
	index := len(p.args)
	p.args = append(p.args, arg)
	return index
}

// Inits returns all the initializers.
func (p *Processor) Inits() []Initializer {
	return p.inits
}

// Args returns the graph arguments.
func (p *Processor) Args() []Argument {
	return p.args
}

// ProcessInits calls all the initializers callbacks.
func (p *Processor) ProcessInits(fc *elements.InputValues) error {
	for _, init := range p.inits {
		if err := init.Init(fc); err != nil {
			return err
		}
	}
	return nil
}
