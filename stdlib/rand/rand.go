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
	"github.com/gx-org/gx/internal/interp/numbers"
	"github.com/gx-org/gx/stdlib/builtin"
)

//go:embed *.gx
var fs embed.FS

// Package description of the GX rand package.
var Package = builtin.PackageBuilder{
	FullPath: "rand",
	Builders: []builtin.Builder{
		builtin.BuildConst(func(pkg *ir.Package) (string, ir.Expr, ir.Type, error) {
			_, value := numbers.NewFloatIR(float64(1<<64), ir.Float64Type())
			return "rescaleRandFloat64", value, value.Type(), nil
		}),
		builtin.BuildConst(func(pkg *ir.Package) (string, ir.Expr, ir.Type, error) {
			_, value := numbers.NewFloatIR(math.Nextafter(1, 0), ir.Float64Type())
			return "maxFloat64BelowOne", value, value.Type(), nil
		}),
		builtin.ParseSource("philox.gx"),
		builtin.ParseSource("rand.gx"),
		builtin.ImplementBuiltin("newBootstrapGenerator", evalNewBootstrapGenerator),
		builtin.ImplementBuiltin("bootstrapGenerator.next", evalBootstrapGeneratorNext),
		builtin.ImplementBuiltin("Philox.Uint32", evalPhiloxUint32),
		builtin.ImplementBuiltin("Philox.Uint64", evalPhiloxUint64),
	},
}
