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
	"fmt"
	"go/ast"
	"math/rand"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/grapheval"
	"github.com/gx-org/gx/interp/proxies"
)

type randBootstrap struct {
	context evaluator.Context
	call    elements.CallAt
	errF    fmterr.FileSet

	seed elements.NumericalElement
	rand *rand.Rand
	next func() (elements.NumericalElement, error)
}

var _ elements.Copier = (*randBootstrap)(nil)

func (rb *randBootstrap) Type() ir.Type {
	return &ir.BuiltinType{Impl: rb}
}

func (rb *randBootstrap) Flatten() ([]elements.Element, error) {
	return []elements.Element{rb}, nil
}

func (rb *randBootstrap) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	return nil, fmterr.Internal(errors.Errorf("%T does not support converting device handles into GX values", rb))
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

func (rb *randBootstrap) nextConstant() (elements.NumericalElement, error) {
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
	return rb.context.Evaluation().Evaluator().ElementFromAtom(elements.NewExprAt(rb.context.File(), expr), value)
}

type randBootstrapArg struct {
	seed   elements.ElementWithArrayFromContext
	ctx    ir.Evaluator
	rb     *randBootstrap
	pValue *proxies.Array
}

func newRandBootstrapArg(ctx evaluator.Context, rb *randBootstrap, seed elements.ElementWithArrayFromContext) (*randBootstrapArg, error) {
	typ := ir.TypeFromKind(ir.Uint64Kind)
	shape := &shape.Shape{DType: dtype.Uint64}
	pValue, err := proxies.NewArray(typ, shape)
	if err != nil {
		return nil, err
	}
	argFactory := &randBootstrapArg{
		rb:     rb,
		ctx:    ctx,
		seed:   seed,
		pValue: pValue,
	}
	ctx.Evaluation().Evaluator().Processor().RegisterInit(argFactory)
	return argFactory, nil
}

func (arg *randBootstrapArg) next() (elements.NumericalElement, error) {
	ev := arg.ctx.(evaluator.Context).Evaluation().Evaluator().(*grapheval.Evaluator)
	src := &ast.Ident{
		Name:    fmt.Sprintf("%T", arg),
		NamePos: arg.rb.call.Node().Source().Pos(),
	}
	return ev.NewArrayArgument(arg, elements.NewExprAt(arg.rb.call.File(), &ir.ValueRef{
		Src: src,
		Stor: &ir.LocalVarStorage{
			Src: src,
			Typ: arg.pValue.Type(),
		},
	}), arg.pValue)
}

func (arg *randBootstrapArg) Init(ctx *elements.InputValues) error {
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

func (arg randBootstrapArg) ValueProxy() proxies.Value {
	return arg.pValue
}

func (arg randBootstrapArg) ValueFromContext(ctx *elements.InputValues) (values.Value, error) {
	val := arg.rb.rand.Uint64()
	return values.AtomIntegerValue[uint64](arg.ValueProxy().Type(), val)
}

func evalNewBootstrapGenerator(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element) ([]elements.Element, error) {
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
	return []elements.Element{elements.NewNamedType(
		ctx.Evaluation().Evaluator().NewFunc,
		call.Node().Type().(*ir.NamedType),
		bootstrap,
	)}, nil
}

func evalBootstrapGeneratorNext(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element) ([]elements.Element, error) {
	bootStrap := elements.Underlying(fn.Recv().Element).(*randBootstrap)
	el, err := bootStrap.next()
	if err != nil {
		return nil, err
	}
	return []elements.Element{el}, nil
}
