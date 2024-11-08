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

// Package interp evaluates GX code to build a computation graph.
//
// The GX Interpreter evaluates GX code represented as an
// intermediate representation (IR) tree
// (see [google3/third_party/gxlang/gx/build/ir/ir]),
// evaluates a function given a backend and static parameters.
//
// The operation of the GX interpreter are implemented in this package,
// excluding backend operations. The operations of the GX interpreter operates on the state of
// the interpreter provided by the
// [google3/third_party/gxlang/gx/interp/state/state] package.
// The state is composed of elements of type [google3/third_party/gxlang/gx/interp/state/state.Element]
// which can be anything that a value can be (scalars, arrays,
// instance of a structure, ...)
//
// Backend operations include the +, -, *, / operators or functions
// provided by the graph constructor (see [google3/third_party/gxlang/backend/graph.Graph]).
// The interpreter will call these function during the interpretation phase
// which, typically, will construct a backend compute graph composed
// of compute node of type [google3/third_party/gxlang/backend/graph.Node].
//
// Some state elements can encapsulate graph nodes (see [google3/third_party/gxlang/gx/interp/state/state.BackendNode])
// or can be converted to graph nodes (e.g. [google3/third_party/gxlang/gx/interp/state/state.Tuple]) but other
// state element cannot be stored in a compute graph (e.g. [google3/third_party/gxlang/gx/interp/state/state.Function]).
package interp

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/backend"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	goplatform "github.com/gx-org/gx/golang/backend/platform"
	"github.com/gx-org/gx/interp/state/proxies"
	"github.com/gx-org/gx/interp/state"
)

type (
	// FuncBuiltin of a builtin function by a backend.
	FuncBuiltin func(ctx Context, call state.CallAt, fn *state.Func, irFunc *ir.FuncBuiltin, args []state.Element) (output state.Element, err error)

	// Interpreter evaluates GX to build a XLA graph.
	Interpreter struct {
		backend        backend.Backend
		packageOptions map[string][]packageOption
		localDevice    *goplatform.Device
	}
)

// New returns a new interpreter.
func New(bck backend.Backend, options []PackageOption) (*Interpreter, error) {
	plat := goplatform.New()
	itrp := &Interpreter{
		backend:        bck,
		packageOptions: make(map[string][]packageOption),
	}
	var err error
	if itrp.localDevice, err = plat.GoDevice(0); err != nil {
		return nil, errors.Errorf("cannot create local evaluator: %v", err)
	}
	if err := itrp.processOptions(options); err != nil {
		return nil, err
	}
	return itrp, nil
}

// Eval evaluates a function.
func (itrp *Interpreter) Eval(fn *ir.FuncDecl, receiver values.Value, args []values.Value) (stat *state.State, out state.Element, err error) {
	if fn.Body == nil {
		return nil, nil, errors.Errorf("%s: missing function body", fn.Name())
	}
	defer func() {
		if err != nil {
			recvName := ""
			recv := fn.FType.Receiver
			if recv != nil {
				recvName = recv.NameT + "."
			}
			err = fmt.Errorf("%s.%s%s evaluation error:\n%w", fn.File().Package.FullName(), recvName, fn.Name(), err)
		}
		err = fmterr.ToStackTraceError(err)
	}()
	funcName := fn.FullyQualifiedName()
	bckGraph := itrp.backend.NewGraph(funcName)
	stat = state.New(fn, bckGraph)
	ctx := newContext(itrp, stat, fn, receiver, args)
	// Add the receiver and arguments to the context.
	frame, err := ctx.pushFuncFrame(fn)
	if err != nil {
		return nil, nil, err
	}
	for _, resultName := range fieldNames(fn.FType.Results.List) {
		frame.define(resultName.Name, nil)
	}
	if receiver != nil {
		receiverProxy, err := proxies.ToProxy(receiver, fn.FType.Receiver)
		if err != nil {
			return nil, nil, err
		}
		field := fn.ReceiverField()
		fieldAt := nodeAt[*ir.Field](ctx, field)
		receiverNode := ctx.state.Receiver(fieldAt, receiverProxy)
		frame.define(field.Name.Name, receiverNode)
	}
	proxyArgs, err := proxies.ToProxies(args, fn.FType.Params.Fields())
	if err != nil {
		return nil, nil, err
	}
	for i, param := range fn.FType.Params.Fields() {
		if i >= len(args) {
			missingParams := fn.FType.Params.Fields()[len(args):]
			builder := strings.Builder{}
			for n, param := range missingParams {
				if n > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(param.Name.String())
			}
			return nil, nil, errors.Errorf("missing parameter(s): %s", builder.String())
		}
		fieldAt := nodeAt[*ir.Field](ctx, param)
		argNode := ctx.state.ArgGX(fieldAt, i, proxyArgs)
		frame.define(param.Name.Name, argNode)
	}
	defer ctx.popFrame()
	out, err = evalFuncBody(ctx, fn.Body)
	return
}

func toSingleNode(ctx *context, stmt *ir.ReturnStmt, elements []state.Element) state.Element {
	if len(elements) == 1 {
		return elements[0]
	}
	return ctx.state.Tuple(ctx.currentFile(), stmt, elements)
}

func rankOf(ctx *context, src ir.SourceNode, typ *ir.ArrayType) (*ir.Rank, error) {
	switch rank := typ.RankF.(type) {
	case *ir.Rank:
		return rank, nil
	case *ir.GenericRank:
		if rank.Rnk == nil {
			return nil, ctx.FileSet().Errorf(src.Source(), "array rank has not been resolved")
		}
		return rank.Rnk, nil
	default:
		return nil, ctx.FileSet().Errorf(src.Source(), "rank %T not supported", rank)
	}
}
