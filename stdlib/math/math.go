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
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/builtin"
)

//go:embed *.gx
var fs embed.FS

// Package description of the GX num package.
var Package = builtin.PackageBuilder{
	FullPath: "math",
	Builders: []builtin.Builder{
		builtin.BuildConst(func(*ir.Package) (string, ir.Expr, ir.Type, error) {
			value := &ir.Number{
				Src: &ast.BasicLit{
					Kind:  token.INT,
					Value: "4294967295",
				},
			}
			return "MaxUint32", value, value.Type(), nil
		}),
		buildConstScalar("InfFloat32", float32(math.Inf(1))),
		buildConstScalar("NegInfFloat32", float32(math.Inf(-1))),
		buildConstScalar("InfFloat64", math.Inf(1)),
		buildConstScalar("NegInfFloat64", math.Inf(-1)),
		builtin.ParseSource(&fs),
		builtin.BuildFunc(exp{}),
		builtin.BuildFunc(pow{}),
		builtin.BuildFunc(log{}),
		builtin.BuildFunc(minFunc{}),
		builtin.BuildFunc(maxFunc{}),
		builtin.BuildFunc(cos{}),
		builtin.BuildFunc(sin{}),
		builtin.BuildFunc(sqrt{}),
		builtin.BuildFunc(tanh{}),
	},
}

// mainAuxArgsToTypes computes the signature of a function with two arguments.
// The result of the function is computed from the first argument.
// The second argument is either a scalar which needs to be of the same numerical type than the first argument
// or an array which needs to be of the same shape than the first argument.
func mainAuxArgsToTypes(funcName string, fetcher ir.Fetcher, call *ir.CallExpr) (main, aux ir.Type, result ir.Type, err error) {
	if len(call.Args) != 2 {
		return nil, nil, nil, fmterr.Errorf(fetcher.FileSet(), call.Source(), "wrong number of arguments in call to %s: got %d but want 2", funcName, len(call.Args))
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
		return nil, nil, nil, fmterr.Internalf(fetcher.FileSet(), call.Source(), "cannot compare arguments type: %v", err)
	}
	if !kindEq {
		return nil, nil, nil, fmterr.Errorf(fetcher.FileSet(), call.Source(), "mismatch types %s and %s", baseType.String(), auxType.String())
	}
	_, auxArrayOk := auxType.(*ir.ArrayType)
	if !auxArrayOk {
		return baseType, auxType, baseType, nil
	}
	shapeEq, err := baseType.Equal(fetcher, auxType)
	if err != nil {
		return nil, nil, nil, fmterr.Internalf(fetcher.FileSet(), call.Source(), "cannot compare arguments type: %v", err)
	}
	if !shapeEq {
		return nil, nil, nil, fmterr.Errorf(fetcher.FileSet(), call.Source(), "mismatch types %s and %s", baseType.String(), auxType.String())
	}
	return baseType, auxType, baseType, nil
}

// buildUnaryFuncType computes the signature of a function with one argument and returning the same
// type.
func buildUnaryFuncType(funcName string, fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	if len(call.Args) != 1 {
		return nil, fmterr.Errorf(fetcher.FileSet(), call.Source(), "wrong number of arguments in call to %s: got %d but want 1", funcName, len(call.Args))
	}
	inferredType, _, err := builtins.InferFromNumericalType(fetcher, call, 0, nil)
	if err != nil {
		return nil, err
	}
	return &ir.FuncType{
		Src:     &ast.FuncType{Func: call.Source().Pos()},
		Params:  builtins.Fields(inferredType),
		Results: builtins.Fields(inferredType),
	}, nil
}

func buildConstScalar[T dtype.GoDataType](name string, value T) builtin.Builder {
	kind := ir.KindGeneric[T]()
	return builtin.BuildConst(func(*ir.Package) (string, ir.Expr, ir.Type, error) {
		value := &ir.AtomicValueT[T]{
			Src: &ast.BasicLit{
				Kind:  token.IDENT,
				Value: name,
			},
			Val: value,
			Typ: ir.ScalarTypeK(kind),
		}
		return name, value, value.Type(), nil
	})
}
