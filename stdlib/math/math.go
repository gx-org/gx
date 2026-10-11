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
	"fmt"
	"go/token"
	"math"
	"math/big"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/src/gx/ast"
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
		buildUnary("Abs", func(g compute.Function) unaryFunc { return g.Abs }),
		buildUnary("Ceil", func(g compute.Function) unaryFunc { return g.Ceil }),
		buildUnary("Cos", func(g compute.Function) unaryFunc { return g.Cos }),
		buildUnary("Erf", func(g compute.Function) unaryFunc { return g.Erf }),
		buildUnary("Expm1", func(g compute.Function) unaryFunc { return g.Expm1 }),
		buildUnary("Exp", func(g compute.Function) unaryFunc { return g.Exp }),
		buildUnary("Floor", func(g compute.Function) unaryFunc { return g.Floor }),
		buildUnary("Log1p", func(g compute.Function) unaryFunc { return g.Log1p }),
		buildUnary("Logistic", func(g compute.Function) unaryFunc { return g.Logistic }),
		buildUnary("Log", func(g compute.Function) unaryFunc { return g.Log }),
		buildBinary("Min", func(g compute.Function) binaryFunc { return g.Min }),
		buildBinary("Max", func(g compute.Function) binaryFunc { return g.Max }),
		buildBinary("Pow", func(g compute.Function) binaryFunc { return g.Pow }),
		buildUnary("Round", func(g compute.Function) unaryFunc { return g.Round }),
		buildUnary("Rsqrt", func(g compute.Function) unaryFunc { return g.Rsqrt }),
		buildUnary("Sign", func(g compute.Function) unaryFunc { return g.Sign }),
		buildUnary("Sin", func(g compute.Function) unaryFunc { return g.Sin }),
		buildUnary("Sqrt", func(g compute.Function) unaryFunc { return g.Sqrt }),
		buildUnary("Tanh", func(g compute.Function) unaryFunc { return g.Tanh }),
		builtin.ImplementBuiltin("checkSameOrScalar", checkSameOrScalar),
	},
}

func checkSameOrScalar(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) (_ []ir.Element, err error) {
	ax1, isSlice := args[0].(*elements.Slice)
	if !isSlice {
		return nil, errors.Errorf("cannot convert %T to %s", args[0], reflect.TypeFor[*elements.Slice]().String())
	}
	ax2, isSlice := args[1].(*elements.Slice)
	if !isSlice {
		return nil, errors.Errorf("cannot convert %T to %s", args[1], reflect.TypeFor[*elements.Slice]().String())
	}
	if ax1.Len() == 0 {
		return []ir.Element{ax2}, nil
	}
	if ax2.Len() == 0 {
		return []ir.Element{ax1}, nil
	}
	same, err := cmp.Equal(env.ExprEval(), ax1, ax2)
	if err != nil {
		return nil, err
	}
	if !same {
		ax1S := ir.ExprString(env.ExprEval(), call.Expr(), ax1)
		ax2S := ir.ExprString(env.ExprEval(), call.Expr(), ax2)
		return []ir.Element{args[0]}, ir.CompileErrorF("cannot use array of shape %s as scalar or array of shape %s (expect same shape)", ax1S, ax2S)
	}
	return []ir.Element{ax1}, err
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
		return nil, nil, nil, fmterr.InternalAt(fetcher.File().FileSet(), call.Node(), "cannot compare arguments type: %v", err)
	}
	if !kindEq {
		return nil, nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "mismatch types %s and %s", baseType.ReferString(fetcher.File()), auxType.ReferString(fetcher.File()))
	}
	if auxType.(ir.ArrayType).Rank().IsAtomic() {
		return baseType, auxType, baseType, nil
	}
	shapeEq, err := baseType.Equal(fetcher, auxType)
	if err != nil {
		return nil, nil, nil, fmterr.InternalAt(fetcher.File().FileSet(), call.Node(), "cannot compare arguments type: %v", err)
	}
	if !shapeEq {
		return nil, nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "mismatch types %s and %s", baseType.ReferString(fetcher.File()), auxType.ReferString(fetcher.File()))
	}
	return baseType, auxType, baseType, nil
}

type goFloats interface {
	float32 | float64
}

func buildConstScalar[T goFloats](name string, value T) builtin.Builder {
	kind := irkind.KindGeneric[T]()
	expr := &ir.NumberCastExpr{
		X: &ir.NumberFloat{
			Src: &ast.BasicLit{
				Kind:  token.FLOAT,
				Value: fmt.Sprint(value),
			},
			Val: big.NewFloat(float64(value)),
		},
		Typ: ir.TypeFromKind(kind),
	}
	return builtin.BuildConst(func(*ir.Package) (string, ir.Expr, ir.Type, error) {
		return name, expr, expr.Type(), nil
	})
}

type unaryFunc = func(compute.Value) (compute.Value, error)

func buildUnary(name string, f func(graph compute.Function) unaryFunc) builtin.Builder {
	return builtin.ImplementBuiltin(name, func(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
		mat := builtin.Materialiser(env)
		x, _, err := materialise.Element(mat, args[0])
		if err != nil {
			return nil, err
		}
		unaryF := f(env.Engine().Graph())
		node, err := unaryF(x)
		if err != nil {
			return nil, err
		}
		return materialise.ElementFromNode(env.ExprEval(), mat, node, call.Type())
	})
}

type binaryFunc = func(x, y compute.Value) (compute.Value, error)

func buildBinary(name string, f func(graph compute.Function) binaryFunc) builtin.Builder {
	return builtin.ImplementBuiltin(name, func(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
		mat := builtin.Materialiser(env)
		x, _, err := materialise.Element(mat, args[0])
		if err != nil {
			return nil, err
		}
		y, _, err := materialise.Element(mat, args[1])
		if err != nil {
			return nil, err
		}
		binaryF := f(env.Engine().Graph())
		node, err := binaryF(x, y)
		if err != nil {
			return nil, err
		}
		return materialise.ElementFromNode(env.ExprEval(), mat, node, call.Type())
	})
}
