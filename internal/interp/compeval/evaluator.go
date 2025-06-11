package compeval

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

// CompEval is the evaluator used for compilation evaluation.
type CompEval struct {
	importer evaluator.Importer
}

// NewHostEvaluator returns a new evaluator for the host.
func NewHostEvaluator(importer evaluator.Importer) *CompEval {
	return &CompEval{importer: importer}
}

// NewSub returns a new evaluator given a new array operator implementations.
func (ev *CompEval) NewSub(elements.ArrayOps) evaluator.Evaluator {
	return ev
}

// NewFunc creates a new function given its definition and a receiver.
func (ev *CompEval) NewFunc(fn ir.Func, recv *elements.Receiver) elements.Func {
	if macro, ok := fn.(*ir.Macro); ok {
		return cpevelements.NewMacro(macro, recv)
	}
	return cpevelements.NewFunc(fn, recv)
}

// Processor returns the processor used to process inits and traces for compiled function.
func (ev *CompEval) Processor() *processor.Processor {
	return nil
}

// Importer returns the importer used by the evaluator.
func (ev *CompEval) Importer() evaluator.Importer {
	return ev.importer
}

// ArrayOps returns the implementation used for array operations.
func (ev *CompEval) ArrayOps() elements.ArrayOps {
	return hostArrayOps
}

// ElementFromAtom returns an element from a GX value.
func (ev *CompEval) ElementFromAtom(src elements.ExprAt, val values.Array) (elements.NumericalElement, error) {
	hostValue, err := val.ToHostArray(kernels.Allocator())
	if err != nil {
		return nil, err
	}
	return cpevelements.NewAtom(src, hostValue)
}

// CallFuncLit calls a function literal.
func (ev *CompEval) CallFuncLit(ctx evaluator.Context, ref *ir.FuncLit, args []elements.Element) ([]elements.Element, error) {
	return nil, errors.Errorf("not implemented")
}

// Trace register a call to the trace builtin function.
func (ev *CompEval) Trace(call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element, fc *elements.InputValues) error {
	return errors.Errorf("not implemented")
}
