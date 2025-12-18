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
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp/grapheval"
	"github.com/gx-org/gx/interp"
)

type randBootstrap struct {
	eval *grapheval.Evaluator
	call elements.CallAt

	seed evaluator.NumericalElement
	rand *rand.Rand
	next func(evaluator.Env) (evaluator.NumericalElement, error)
}

var _ elements.Copier = (*randBootstrap)(nil)

func (rb *randBootstrap) Type() ir.Type {
	return &ir.BuiltinType{Impl: rb}
}

func (*randBootstrap) Kind() ir.Kind {
	return ir.InterfaceKind
}

func (rb *randBootstrap) Copy() elements.Copier {
	return rb
}

func (rb *randBootstrap) initRand(seed *values.HostArray) error {
	seedValue := types.AtomFromHost[int64](seed)
	rb.rand = rand.New(rand.NewSource(seedValue))
	return nil
}

var uint64Type = ir.TypeFromKind(ir.Uint64Kind)

func (rb *randBootstrap) nextConstant(env evaluator.Env) (evaluator.NumericalElement, error) {
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
	return rb.eval.ElementFromAtom(env.File(), expr, value)
}

type randBootstrapArg struct {
	rb    *randBootstrap
	seed  elements.ElementWithArrayFromContext
	proxy ir.Element
}

var seedType = ir.Uint64Type()

func newRandBootstrapArg(ctx evaluator.Env, rb *randBootstrap, seed elements.ElementWithArrayFromContext) (*randBootstrapArg, error) {
	argFactory := &randBootstrapArg{
		rb:    rb,
		seed:  seed,
		proxy: cpevelements.NewArray(seedType),
	}
	ctx.Evaluator().Processor().RegisterInit(argFactory)
	return argFactory, nil
}

func (arg *randBootstrapArg) next(env evaluator.Env) (evaluator.NumericalElement, error) {
	return arg.rb.eval.NewArrayArgument(env.File(), arg, seedType, arg.proxy)
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

func (arg *randBootstrapArg) Name() string {
	return "randBootstrapArg.next()"
}

func (arg *randBootstrapArg) ValueFromContext(ctx *values.FuncInputs) (ir.Element, error) {
	val := arg.rb.rand.Uint64()
	return values.AtomIntegerValue[uint64](seedType, val)
}

func (arg *randBootstrapArg) Evaluator() *grapheval.Evaluator {
	return arg.rb.eval
}

func evalNewBootstrapGenerator(ctx evaluator.Env, call elements.CallAt, fn fun.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	bootstrap := &randBootstrap{
		eval: ctx.Evaluator().(*grapheval.Evaluator),
		call: call,
	}
	var err error
	switch seedNode := args[0].(type) {
	case elements.ElementWithConstant:
		bootstrap.next = bootstrap.nextConstant
		var cst *values.HostArray
		cst, err = seedNode.NumericalConstant()
		if err != nil {
			return nil, err
		}
		err = bootstrap.initRand(cst)
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
	return []ir.Element{fun.NewNamedType(
		interp.NewRunFunc,
		call.Node().Type().(*ir.NamedType),
		bootstrap,
	)}, nil
}

func evalBootstrapGeneratorNext(ctx evaluator.Env, call elements.CallAt, fn fun.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	bootStrap := elements.Underlying(fn.Recv().Element).(*randBootstrap)
	el, err := bootStrap.next(ctx)
	if err != nil {
		return nil, err
	}
	return []ir.Element{el}, nil
}
