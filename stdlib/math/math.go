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

// Package math provides math functions in GX.
// Math functions do not interact with shapes,
package math

import (
	"embed"
	"go/ast"
	"go/token"
	"math"

	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/grapheval"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

//go:embed *.gx
var fs embed.FS

// Package description of the GX num package.
var Package = builtin.PackageBuilder{
	FullPath: "math",
	Builders: []builtin.Builder{
		// Insert numeric constants that cannot be expressed as literals:
		buildConstScalar("InfFloat32", float32(math.Inf(1))),
		buildConstScalar("NegInfFloat32", float32(math.Inf(-1))),
		buildConstScalar("InfFloat64", math.Inf(1)),
		buildConstScalar("NegInfFloat64", math.Inf(-1)),
		builtin.ParseSource(&fs),
		builtin.BuildFunc(pow{}),
		builtin.BuildFunc(minFunc{}),
		builtin.BuildFunc(maxFunc{}),
		builtin.ImplementStubFunc("Abs", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Math.Abs }),
		builtin.ImplementStubFunc("Ceil", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Math.Ceil }),
		buildUnary("Cos", func(g ops.Graph) unaryFunc { return g.Math().Cos }),
		builtin.ImplementStubFunc("Erf", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Math.Erf }),
		builtin.ImplementStubFunc("Expm1", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Math.Expm1 }),
		builtin.ImplementStubFunc("Exp", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Math.Exp }),
		builtin.ImplementStubFunc("Floor", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Math.Floor }),
		builtin.ImplementStubFunc("Log1p", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Math.Log1p }),
		builtin.ImplementStubFunc("Logistic", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Math.Logistic }),
		builtin.ImplementStubFunc("Log", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Math.Log }),
		builtin.ImplementStubFunc("Round", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Math.Round }),
		builtin.ImplementStubFunc("Rsqrt", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Math.Rsqrt }),
		builtin.ImplementStubFunc("Sign", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Math.Sign }),
		buildUnary("Sin", func(g ops.Graph) unaryFunc { return g.Math().Sin }),
		builtin.ImplementStubFunc("Sqrt", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Math.Sqrt }),
		buildUnary("Tanh", func(g ops.Graph) unaryFunc { return g.Math().Tanh }),
	},
}

// mainAuxArgsToTypes computes the signature of a function with two arguments.
// The result of the function is computed from the first argument.
// The second argument is either a scalar which needs to be of the same numerical type than the first argument
// or an array which needs to be of the same shape than the first argument.
func mainAuxArgsToTypes(funcName string, fetcher ir.Fetcher, call *ir.CallExpr) (main, aux ir.Type, result ir.Type, err error) {
	if len(call.Args) != 2 {
		return nil, nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "wrong number of arguments in call to %s: got %d but want 2", funcName, len(call.Args))
	}
	baseType, baseNumType, err := builtins.InferFromNumericalType(fetcher, call, 0, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	auxType, auxNumTyp, err := builtins.InferFromNumericalType(fetcher, call, 1, baseNumType)
	if err != nil {
		return nil, nil, nil, err
	}
	kindEq, err := baseNumType.Equal(fetcher, auxNumTyp)
	if err != nil {
		return nil, nil, nil, fmterr.Internalf(fetcher.File().FileSet(), call.Source(), "cannot compare arguments type: %v", err)
	}
	if !kindEq {
		return nil, nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "mismatch types %s and %s", baseType.String(), auxType.String())
	}
	if auxType.(ir.ArrayType).Rank().IsAtomic() {
		return baseType, auxType, baseType, nil
	}
	shapeEq, err := baseType.Equal(fetcher, auxType)
	if err != nil {
		return nil, nil, nil, fmterr.Internalf(fetcher.File().FileSet(), call.Source(), "cannot compare arguments type: %v", err)
	}
	if !shapeEq {
		return nil, nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "mismatch types %s and %s", baseType.String(), auxType.String())
	}
	return baseType, auxType, baseType, nil
}

func buildConstScalar[T dtype.GoDataType](name string, value T) builtin.Builder {
	kind := ir.KindGeneric[T]()
	return builtin.BuildConst(func(*ir.Package) (string, ir.AssignableExpr, ir.Type, error) {
		value := &ir.AtomicValueT[T]{
			Src: &ast.BasicLit{
				Kind:  token.IDENT,
				Value: name,
			},
			Val: value,
			Typ: ir.TypeFromKind(kind),
		}
		return name, value, value.Type(), nil
	})
}

type unaryFunc = func(ops.Node) (ops.Node, error)

func buildUnary(name string, f func(graph ops.Graph) unaryFunc) builtin.Builder {
	return builtin.ImplementGraphFunc(name, func(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element) ([]elements.Element, error) {
		ao := ctx.Evaluation().Evaluator().ArrayOps()
		x, xShape, err := grapheval.NodeFromElement(ao, args[0])
		if err != nil {
			return nil, err
		}
		unaryF := f(ctx.Evaluation().Evaluator().ArrayOps().Graph())
		node, err := unaryF(x)
		if err != nil {
			return nil, err
		}
		return grapheval.ElementsFromNode(call.ToExprAt(), &ops.OutputNode{
			Node:  node,
			Shape: xShape,
		})
	})
}
