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
	"go/ast"
	"go/token"
	"math"

	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/concrete"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
)

// Package description of the GX num package.
var Package = builtin.PackageBuilder{
	FullPath: "math",
	Builders: []builtin.Builder{
		// Insert numeric constants that cannot be expressed as literals:
		buildConstScalar("InfFloat32", float32(math.Inf(1))),
		buildConstScalar("NegInfFloat32", float32(math.Inf(-1))),
		buildConstScalar("InfFloat64", math.Inf(1)),
		buildConstScalar("NegInfFloat64", math.Inf(-1)),
		builtin.ParseSource(),
		builtin.BuildFunc(pow{}),
		builtin.BuildFunc(minFunc{}),
		builtin.BuildFunc(maxFunc{}),
		buildUnary("Abs", func(g ops.Graph) unaryFunc { return g.Math().Abs }),
		buildUnary("Ceil", func(g ops.Graph) unaryFunc { return g.Math().Ceil }),
		buildUnary("Cos", func(g ops.Graph) unaryFunc { return g.Math().Cos }),
		buildUnary("Erf", func(g ops.Graph) unaryFunc { return g.Math().Erf }),
		buildUnary("Expm1", func(g ops.Graph) unaryFunc { return g.Math().Expm1 }),
		buildUnary("Exp", func(g ops.Graph) unaryFunc { return g.Math().Exp }),
		buildUnary("Floor", func(g ops.Graph) unaryFunc { return g.Math().Floor }),
		buildUnary("Log1p", func(g ops.Graph) unaryFunc { return g.Math().Log1p }),
		buildUnary("Logistic", func(g ops.Graph) unaryFunc { return g.Math().Logistic }),
		buildUnary("Log", func(g ops.Graph) unaryFunc { return g.Math().Log }),
		buildUnary("Round", func(g ops.Graph) unaryFunc { return g.Math().Round }),
		buildUnary("Rsqrt", func(g ops.Graph) unaryFunc { return g.Math().Rsqrt }),
		buildUnary("Sign", func(g ops.Graph) unaryFunc { return g.Math().Sign }),
		buildUnary("Sin", func(g ops.Graph) unaryFunc { return g.Math().Sin }),
		buildUnary("Sqrt", func(g ops.Graph) unaryFunc { return g.Math().Sqrt }),
		buildUnary("Tanh", func(g ops.Graph) unaryFunc { return g.Math().Tanh }),
	},
}

// mainAuxArgsToTypes computes the signature of a function with two arguments.
// The result of the function is computed from the first argument.
// The second argument is either a scalar which needs to be of the same numerical type than the first argument
// or an array which needs to be of the same shape than the first argument.
func mainAuxArgsToTypes(funcName string, fetcher ir.Fetcher, call *ir.FuncCallExpr) (main, aux ir.Type, result ir.Type, err error) {
	if len(call.Args) != 2 {
		return nil, nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "wrong number of arguments in call to %s: got %d but want 2", funcName, len(call.Args))
	}
	baseType, baseNumType, err := builtins.InferFromNumericalType(fetcher, call, 0, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	auxType, auxNumTyp, err := builtins.InferFromNumericalType(fetcher, call, 1, baseNumType)
	if err != nil {
		return nil, nil, nil, err
	}
	kindEq, cpErr, err := baseNumType.Equal(fetcher, auxNumTyp)
	if err != nil {
		return nil, nil, nil, fmterr.Internalf(fetcher.File().FileSet(), call.Node(), "cannot compare arguments type: %v", err)
	}
	if cpErr != nil {
		return nil, nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "mismatch types %s and %s: %s", baseType.ReferString(fetcher.File()), auxType.ReferString(fetcher.File()), cpErr.Error())
	}
	if !kindEq {
		return nil, nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "mismatch types %s and %s", baseType.ReferString(fetcher.File()), auxType.ReferString(fetcher.File()))
	}
	if auxType.(ir.ArrayType).Rank().IsAtomic() {
		return baseType, auxType, baseType, nil
	}
	shapeEq, cpErr, err := baseType.Equal(fetcher, auxType)
	if err != nil {
		return nil, nil, nil, fmterr.Internalf(fetcher.File().FileSet(), call.Node(), "cannot compare arguments type: %v", err)
	}
	if cpErr != nil {
		return nil, nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "mismatch types %s and %s: %s", baseType.ReferString(fetcher.File()), auxType.ReferString(fetcher.File()), cpErr.Error())
	}
	if !shapeEq {
		return nil, nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "mismatch types %s and %s", baseType.ReferString(fetcher.File()), auxType.ReferString(fetcher.File()))
	}
	return baseType, auxType, baseType, nil
}

func buildConstScalar[T dtype.GoDataType](name string, value T) builtin.Builder {
	kind := irkind.KindGeneric[T]()
	return builtin.BuildConst(func(*ir.Package) (string, ir.Expr, ir.Type, error) {
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
	return builtin.ImplementGraphFunc(name, func(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
		mat := builtin.Materialiser(env)
		x, xShape, err := materialise.Element(mat, args[0])
		if err != nil {
			return nil, err
		}
		unaryF := f(env.Engine().ArrayOps().Graph())
		node, err := unaryF(x)
		if err != nil {
			return nil, err
		}
		typ, cpErr, err := concrete.Concrete(env.ExprEval(), call.Expr(), call.Type())
		if unErr := ir.UnifyErr(cpErr, err); err != nil {
			return nil, unErr
		}
		return materialise.ElementFromNode(env.File(), mat, &ops.OutputNode{
			Node:  node,
			Shape: xShape,
		}, typ)
	})
}
