// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rand

import (
	"math/rand"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/grapheval"
	"github.com/gx-org/gx/interp"
)

type randBootstrap struct {
	context evaluator.Context
	call    elements.CallAt
	errF    fmterr.FileSet

	seed evaluator.NumericalElement
	rand *rand.Rand
	next func(*interp.FileScope) (evaluator.NumericalElement, error)
}

var _ interp.Copier = (*randBootstrap)(nil)

func (rb *randBootstrap) Type() ir.Type {
	return &ir.BuiltinType{Impl: rb}
}

func (*randBootstrap) Kind() ir.Kind {
	return ir.InterfaceKind
}

func (rb *randBootstrap) Copy() interp.Copier {
	return rb
}

func (rb *randBootstrap) initRand(seed *values.HostArray) error {
	seedValue := types.AtomFromHost[int64](seed)
	rb.rand = rand.New(rand.NewSource(seedValue))
	return nil
}

var uint64Type = ir.TypeFromKind(ir.Uint64Kind)

func (rb *randBootstrap) nextConstant(*interp.FileScope) (evaluator.NumericalElement, error) {
	next := rb.rand.Uint64()
	expr := &ir.AtomicValueT[uint64]{
		Src: rb.call.Node().Expr(),
		Val: next,
		Typ: uint64Type,
	}
	value, err := values.AtomIntegerValue(expr.Typ, next)
	if err != nil {
		return nil, err
	}
	return rb.context.Evaluator().ElementFromAtom(rb.context, expr, value)
}

type randBootstrapArg struct {
	seed  elements.ElementWithArrayFromContext
	ctx   ir.Evaluator
	rb    *randBootstrap
	proxy ir.Element
}

var seedType = ir.Uint64Type()

func newRandBootstrapArg(ctx evaluator.Context, rb *randBootstrap, seed elements.ElementWithArrayFromContext) (*randBootstrapArg, error) {
	argFactory := &randBootstrapArg{
		rb:    rb,
		ctx:   ctx,
		seed:  seed,
		proxy: cpevelements.NewArray(seedType),
	}
	ctx.Evaluator().Processor().RegisterInit(argFactory)
	return argFactory, nil
}

func (arg *randBootstrapArg) next(fitp *interp.FileScope) (evaluator.NumericalElement, error) {
	ev := arg.ctx.(evaluator.Context).Evaluator().(*grapheval.Evaluator)
	return ev.NewArrayArgument(fitp, arg, seedType, arg.proxy)
}

func (arg *randBootstrapArg) Init(ctx *values.FuncInputs) error {
	value, err := arg.seed.ArrayFromContext(ctx)
	if err != nil {
		return nil
	}
	hostValue, err := value.ToHost(kernels.Allocator())
	if err != nil {
		return err
	}
	array, ok := hostValue.(*values.HostArray)
	if !ok {
		return errors.Errorf("cannot convert GX argument %T to %T: not supported", value, array)
	}
	return arg.rb.initRand(array)
}

func (arg randBootstrapArg) Name() string {
	return "randBootstrapArg.next()"
}

func (arg randBootstrapArg) ValueFromContext(ctx *values.FuncInputs) (ir.Element, error) {
	val := arg.rb.rand.Uint64()
	return values.AtomIntegerValue[uint64](seedType, val)
}

func evalNewBootstrapGenerator(ctx evaluator.Context, call elements.CallAt, fn interp.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	bootstrap := &randBootstrap{
		context: ctx,
		call:    call,
		errF:    fmterr.FileSet{FSet: ctx.File().FileSet()},
	}
	var err error
	switch seedNode := args[0].(type) {
	case elements.ElementWithConstant:
		bootstrap.next = bootstrap.nextConstant
		err = bootstrap.initRand(seedNode.NumericalConstant())
	case elements.ElementWithArrayFromContext:
		var argFactory *randBootstrapArg
		argFactory, err = newRandBootstrapArg(ctx, bootstrap, seedNode)
		if err != nil {
			return nil, err
		}
		bootstrap.next = argFactory.next
	default:
		err = errors.Errorf("cannot process seed node: %T not supported", seedNode)
	}
	if err != nil {
		return nil, err
	}
	return []ir.Element{interp.NewNamedType(
		ctx.(*interp.FileScope).NewFunc,
		call.Node().Type().(*ir.NamedType),
		bootstrap,
	)}, nil
}

func evalBootstrapGeneratorNext(ctx evaluator.Context, call elements.CallAt, fn interp.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	bootStrap := interp.Underlying(fn.Recv().Element).(*randBootstrap)
	el, err := bootStrap.next(ctx.(*interp.FileScope))
	if err != nil {
		return nil, err
	}
	return []ir.Element{el}, nil
}
