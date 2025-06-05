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

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

//go:embed *.gx
var fs embed.FS

// Package description of the GX rand package.
var Package = builtin.PackageBuilder{
	FullPath: "rand",
	Builders: []builtin.Builder{
		builtin.BuildConst(func(pkg *ir.Package) (string, ir.AssignableExpr, ir.Type, error) {
			value := &ir.AtomicValueT[float64]{
				Src: pkg.Name,
				Val: float64(1 << 64),
				Typ: ir.TypeFromKind(ir.Float64Kind),
			}
			return "rescaleRandFloat64", value, value.Type(), nil
		}),
		builtin.BuildConst(func(pkg *ir.Package) (string, ir.AssignableExpr, ir.Type, error) {
			value := &ir.AtomicValueT[float64]{
				Src: pkg.Name,
				Val: math.Nextafter(1, 0),
				Typ: ir.TypeFromKind(ir.Float64Kind),
			}
			return "maxFloat64BelowOne", value, value.Type(), nil
		}),
		builtin.ParseSource(&fs, "philox.gx"),
		builtin.ParseSource(&fs, "rand.gx"),
		builtin.ImplementBuiltin("newBootstrapGenerator", evalNewBootstrapGenerator),
		builtin.ImplementBuiltin("bootstrapGenerator.next", evalBootstrapGeneratorNext),
		builtin.ImplementStubFunc("Philox.Uint32", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Rand.PhiloxUint32 }),
		builtin.ImplementStubFunc("Philox.Uint64", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Rand.PhiloxUint64 }),
	},
}
