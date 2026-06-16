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
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/concrete"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
	gxerrors "github.com/gx-org/gx/stdlib/errors"
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
		buildBinary("Min", func(g ops.Graph) binaryFunc { return g.Math().Min }),
		buildBinary("Max", func(g ops.Graph) binaryFunc { return g.Math().Max }),
		buildBinary("Pow", func(g ops.Graph) binaryFunc { return g.Math().Pow }),
		buildUnary("Round", func(g ops.Graph) unaryFunc { return g.Math().Round }),
		buildUnary("Rsqrt", func(g ops.Graph) unaryFunc { return g.Math().Rsqrt }),
		buildUnary("Sign", func(g ops.Graph) unaryFunc { return g.Math().Sign }),
		buildUnary("Sin", func(g ops.Graph) unaryFunc { return g.Math().Sin }),
		buildUnary("Sqrt", func(g ops.Graph) unaryFunc { return g.Math().Sqrt }),
		buildUnary("Tanh", func(g ops.Graph) unaryFunc { return g.Math().Tanh }),
		builtin.ImplementBuiltin("checkSameOrScalar", checkSameOrScalar),
	},
}

func checkSameOrScalar(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) (_ []ir.Element, err error) {
	ax1, isSlice := args[0].(*elements.Slice)
	if !isSlice {
		return nil, errors.Errorf("cannot convert %T to %s", args[0], reflect.TypeFor[*elements.Slice]().String())
	}
	ax2, isSlice := args[1].(*elements.Slice)
	if !isSlice {
		return nil, errors.Errorf("cannot convert %T to %s", args[1], reflect.TypeFor[*elements.Slice]().String())
	}
	if ax1.Len() == 0 {
		return []ir.Element{ax2, elements.NilError()}, nil
	}
	if ax2.Len() == 0 {
		return []ir.Element{ax1, elements.NilError()}, nil
	}
	same, err := ax1.Compare(ax2)
	if err != nil {
		return nil, err
	}
	if !same {
		ax1S := ir.ExprString(env.ExprEval(), call.Expr(), ax1)
		ax2S := ir.ExprString(env.ExprEval(), call.Expr(), ax2)
		gxErr, err := gxerrors.Errorf(env, "cannot use array of shape %s as scalar or array of shape %s (expect same shape)", ax1S, ax2S)
		return []ir.Element{args[0], gxErr}, err
	}
	return []ir.Element{ax1, elements.NilError()}, err
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
	kindEq, err := baseNumType.Equal(fetcher, auxNumTyp)
	if err != nil {
		return nil, nil, nil, fmterr.Internalf(fetcher.File().FileSet(), call.Node(), "cannot compare arguments type: %v", err)
	}
	if !kindEq {
		return nil, nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "mismatch types %s and %s", baseType.ReferString(fetcher.File()), auxType.ReferString(fetcher.File()))
	}
	if auxType.(ir.ArrayType).Rank().IsAtomic() {
		return baseType, auxType, baseType, nil
	}
	shapeEq, err := baseType.Equal(fetcher, auxType)
	if err != nil {
		return nil, nil, nil, fmterr.Internalf(fetcher.File().FileSet(), call.Node(), "cannot compare arguments type: %v", err)
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
	return builtin.ImplementBuiltin(name, func(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
		args = args[call.Callee.FuncType().Origin().TypeParams.Len():]
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
		typ, err := concrete.Concrete(env.ExprEval(), call.Expr(), call.Type())
		if err != nil {
			return nil, err
		}
		return materialise.ElementFromNode(env.File(), mat, &ops.OutputNode{
			Node:  node,
			Shape: xShape,
		}, typ)
	})
}

type binaryFunc = func(x, y ops.Node) (ops.Node, error)

func buildBinary(name string, f func(graph ops.Graph) binaryFunc) builtin.Builder {
	return builtin.ImplementBuiltin(name, func(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
		args = args[call.Callee.FuncType().Origin().TypeParams.Len():]
		mat := builtin.Materialiser(env)
		x, xShape, err := materialise.Element(mat, args[0])
		if err != nil {
			return nil, err
		}
		y, _, err := materialise.Element(mat, args[1])
		if err != nil {
			return nil, err
		}
		binaryF := f(env.Engine().ArrayOps().Graph())
		node, err := binaryF(x, y)
		if err != nil {
			return nil, err
		}
		typ, err := concrete.Concrete(env.ExprEval(), call.Expr(), call.Type())
		if err != nil {
			return nil, err
		}
		return materialise.ElementFromNode(env.File(), mat, &ops.OutputNode{
			Node:  node,
			Shape: xShape,
		}, typ)
	})
}
