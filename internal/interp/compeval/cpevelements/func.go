package cpevelements

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

type fun struct {
	fn   ir.Func
	recv *elements.Receiver
}

// NewFunc creates a new function given its definition and a receiver.
func NewFunc(fn ir.Func, recv *elements.Receiver) elements.Func {
	return &fun{fn: fn, recv: recv}
}

func (f *fun) Func() ir.Func {
	return f.fn
}

func (f *fun) Recv() *elements.Receiver {
	return f.recv
}

func (f *fun) Call(fctx elements.FileContext, call *ir.CallExpr, args []elements.Element) ([]elements.Element, error) {
	ctx := fctx.(evaluator.Context)
	res := call.Callee.T.Results.Fields()
	els := make([]elements.Element, len(res))
	for i, ri := range res {
		var err error
		els[i], err = NewRuntimeValue(ctx, &ir.ValueRef{
			Stor: &ir.LocalVarStorage{
				Typ: ri.Type(),
			},
		})
		if err != nil {
			return nil, err
		}
	}
	return els, nil
}

func (f *fun) Flatten() ([]elements.Element, error) {
	return []elements.Element{f}, nil
}

func (f *fun) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	return nil, errors.Errorf("not implemented")
}

func (f *fun) Kind() ir.Kind {
	return ir.FuncKind
}

// FuncDeclFromElement extracts a function declaration from an element.
func FuncDeclFromElement(el elements.Element) (*ir.FuncDecl, error) {
	fEl, ok := el.(elements.Func)
	if !ok {
		return nil, errors.Errorf("cannot convert element %T to a function", el)
	}
	fun := fEl.Func()
	fDecl, ok := fun.(*ir.FuncDecl)
	if !ok {
		return nil, errors.Errorf("%s is not a GX user function", fun.Name())
	}
	return fDecl, nil
}
