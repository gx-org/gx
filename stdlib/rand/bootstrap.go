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
	"github.com/gx-org/backend/dtypes"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/internal/interp/numbers"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/grapheval"
	"github.com/gx-org/gx/interp"
)

type randBootstrap struct {
	eval *grapheval.Evaluator
	call *ir.FuncCallExpr

	seed engine.NumericalElement
	rand *rand.Rand
	next func(*engine.Env) (engine.NumericalElement, error)
}

var _ engine.Copier = (*randBootstrap)(nil)

func (rb *randBootstrap) Type() ir.Type {
	return &ir.BuiltinType{Impl: rb}
}

func (*randBootstrap) Kind() irkind.Kind {
	return irkind.Interface
}

func (rb *randBootstrap) Copy() engine.Copier {
	return rb
}

func (rb *randBootstrap) initRand(seed int64) {
	rb.rand = rand.New(rand.NewSource(seed))
}

func (rb *randBootstrap) nextConstant(env *engine.Env) (engine.NumericalElement, error) {
	cstUint64 := rb.rand.Uint64()
	return numbers.NewElement(env, ir.Uint64Type(), cstUint64)
}

type randBootstrapArg struct {
	rb    *randBootstrap
	seed  elements.ElementWithArrayFromContext
	proxy ir.Element
}

var (
	seedType  = ir.Uint64Type()
	seedShape = &shape.Shape{
		DType: dtypes.Uint64,
	}
)

func newRandBootstrapArg(env *engine.Env, rb *randBootstrap, seed elements.ElementWithArrayFromContext) (*randBootstrapArg, error) {
	argFactory := &randBootstrapArg{
		rb:   rb,
		seed: seed,
	}
	env.Engine().Processor().RegisterInit(argFactory)
	return argFactory, nil
}

func (arg *randBootstrapArg) next(env *engine.Env) (engine.NumericalElement, error) {
	return arg.rb.eval.NewArrayArgument(env.File(), arg, seedType, seedShape)
}

func (arg *randBootstrapArg) Init(ctx *hostio.FuncInputs) error {
	value, err := arg.seed.ArrayFromContext(ctx)
	if err != nil {
		return nil
	}
	hostValue, err := value.ToHost(kernels.Allocator())
	if err != nil {
		return err
	}
	array, ok := hostValue.(*hostio.HostArray)
	if !ok {
		return errors.Errorf("cannot convert GX argument %T to %T: not supported", value, array)
	}
	val, err := hostio.ToAtom[int64](array)
	arg.rb.initRand(val)
	return err
}

func (arg *randBootstrapArg) Name() string {
	return "randBootstrapArg.next()"
}

func (arg *randBootstrapArg) ValueFromContext(ctx *hostio.FuncInputs) (ir.Element, error) {
	val := arg.rb.rand.Uint64()
	return hostio.AtomIntegerValue[uint64](seedType, val)
}

func (arg *randBootstrapArg) Evaluator() *grapheval.Evaluator {
	return arg.rb.eval
}

func evalNewBootstrapGenerator(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	bootstrap := &randBootstrap{
		eval: env.Engine().(*grapheval.Evaluator),
		call: call,
	}
	var err error
	switch seedNode := args[0].(type) {
	case engine.ConstantElement:
		bootstrap.next = bootstrap.nextConstant
		seed, err := elements.Int64FromElement(seedNode)
		if err != nil {
			return nil, err
		}
		bootstrap.initRand(seed)
	case elements.ElementWithArrayFromContext:
		var argFactory *randBootstrapArg
		argFactory, err = newRandBootstrapArg(env, bootstrap, seedNode)
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
	return []ir.Element{elements.NewNamedType(
		interp.NewRunFunc,
		call.Type().(*ir.NamedType),
		bootstrap,
	)}, nil
}

func evalBootstrapGeneratorNext(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	under, err := coreiface.Underlying(recv)
	if err != nil {
		return nil, err
	}
	bootStrap, err := cast.To[*randBootstrap](under)
	if err != nil {
		return nil, err
	}
	el, err := bootStrap.next(env)
	if err != nil {
		return nil, err
	}
	return []ir.Element{el}, nil
}
