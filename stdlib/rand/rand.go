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

// Package rand provides the functions in the rand GX standard library.
package rand

import (
	"embed"
	"math"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

//go:embed *.gx
var fs embed.FS

// Package description of the GX rand package.
var Package = builtin.PackageBuilder{
	FullPath: "rand",
	Builders: []builtin.Builder{
		builtin.BuildConst(func(pkg *ir.Package) (string, ir.Expr, ir.Type, error) {
			value := &ir.AtomicValueT[float64]{
				Src: pkg.Name,
				Val: float64(1 << 64),
				Typ: ir.TypeFromKind(ir.Float64Kind),
			}
			return "rescaleRandFloat64", value, value.Type(), nil
		}),
		builtin.BuildConst(func(pkg *ir.Package) (string, ir.Expr, ir.Type, error) {
			value := &ir.AtomicValueT[float64]{
				Src: pkg.Name,
				Val: math.Nextafter(1, 0),
				Typ: ir.TypeFromKind(ir.Float64Kind),
			}
			return "maxFloat64BelowOne", value, value.Type(), nil
		}),
		builtin.BuildType(bootstrapGenerator{}),
		builtin.BuildFunc(newBootstrapGenerator{}),
		builtin.ParseSource(&fs, "philox.gx"),
		builtin.BuildMethod("Philox", philoxUint32{}),
		builtin.BuildMethod("Philox", philoxUint64{}),
		builtin.ParseSource(&fs, "rand.gx"),
	},
}

type bootstrapGeneratorNext struct {
	builtin.Func
}

func (bootstrapGeneratorNext) buildFuncIR(impl *impl.Stdlib, pkg *ir.Package, generatorType *ir.NamedType) *ir.FuncBuiltin {
	resultType := ir.TypeFromKind(ir.Uint64Kind)
	fn := &ir.FuncBuiltin{
		FName: "next",
		FType: &ir.FuncType{
			Receiver: generatorType,
			Params:   &ir.FieldList{},
			Results: &ir.FieldList{
				List: []*ir.FieldGroup{
					&ir.FieldGroup{
						Type: resultType,
					},
				},
			},
		},
	}
	fn.Impl = bootstrapGeneratorNext{Func: builtin.Func{
		Func: fn,
		Impl: impl.Rand.BootstrapGeneratorNext,
	}}
	return fn
}

func (f bootstrapGeneratorNext) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return f.Func.Func.FType, nil
}

type bootstrapGenerator struct {
	builtin *ir.BuiltinType
	typ     *ir.NamedType
}

const bootstrapGeneratorTypeName = "bootstrapGenerator"

func (bootstrapGenerator) BuildNamedType(impl *impl.Stdlib, pkg *ir.Package) (*ir.NamedType, error) {
	builtin := &ir.BuiltinType{}
	generatorType := &ir.NamedType{
		NameT:      bootstrapGeneratorTypeName,
		Underlying: builtin,
	}
	builtin.Impl = bootstrapGenerator{
		builtin: builtin,
		typ:     generatorType,
	}
	next := bootstrapGeneratorNext{}.buildFuncIR(impl, pkg, generatorType)
	generatorType.Methods = append(generatorType.Methods, next)
	return generatorType, nil
}

type newBootstrapGenerator struct {
	builtin.Func
}

func (newBootstrapGenerator) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	generatorType := pkg.TypeByName(bootstrapGeneratorTypeName)
	if generatorType == nil {
		return nil, errors.Errorf("builtin type %s undefined", bootstrapGeneratorTypeName)
	}
	fn := &ir.FuncBuiltin{
		FName: "newBootstrapGenerator",
		FType: &ir.FuncType{
			Params: &ir.FieldList{
				List: []*ir.FieldGroup{
					&ir.FieldGroup{
						Type: ir.TypeFromKind(ir.Int64Kind),
					},
				},
			},
			Results: &ir.FieldList{
				List: []*ir.FieldGroup{
					&ir.FieldGroup{
						Type: generatorType,
					},
				},
			},
		},
	}
	fn.Impl = newBootstrapGenerator{Func: builtin.Func{
		Func: fn,
		Impl: impl.Rand.BootstrapGeneratorNew,
	}}
	return fn, nil
}

func (f newBootstrapGenerator) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return f.Func.Func.FType, nil
}
