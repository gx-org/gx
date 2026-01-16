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

// Package builtins provides builtins for the compiler.
package builtins

import (
	"go/token"
	"maps"

	"github.com/gx-org/gx/build/ir"
)

// Registerer is a function registering a builtin.
type Registerer func(token.Token, ir.Storage)

var irBuiltins = map[string]func(Registerer){}

func init() {
	// Builtin types.
	registerBuiltinType("any", ir.AnyType())
	registerBuiltinType("bool", ir.BoolType())
	registerBuiltinType("bfloat16", ir.Bfloat16Type())
	registerBuiltinType("float32", ir.Float32Type())
	registerBuiltinType("float64", ir.Float64Type())
	registerBuiltinType("int32", ir.Int32Type())
	registerBuiltinType("int64", ir.Int64Type())
	registerBuiltinType("string", ir.StringType())
	registerBuiltinType("uint32", ir.Uint32Type())
	registerBuiltinType("uint64", ir.Uint64Type())
	registerBuiltinType("intlen", ir.IntLenType())
	registerBuiltinType("intidx", ir.IntIndexType())

	// Builtin values.
	registerBuiltinIR(token.CONST, ir.FalseStorage())
	registerBuiltinIR(token.CONST, ir.TrueStorage())

	// Builtin functions.
	registerBuiltinFunc(Append())
	registerBuiltinFunc(AxLengths())
	registerBuiltinFunc(Set())
	registerBuiltinFunc(Trace())
}

func registerBuiltinIR(tok token.Token, store ir.Storage) {
	irBuiltins[store.NameDef().Name] = func(reg Registerer) {
		reg(tok, store)
	}
}

func registerBuiltinType(name string, typ ir.Type) {
	registerBuiltinIR(token.TYPE, ir.BuiltinStorage(name, ir.TypeExpr(nil, typ)))
}

func registerBuiltinFunc(impl ir.FuncImpl) {
	registerBuiltinIR(token.FUNC, ir.BuiltinFunction(impl))
}

// Register all the builtins.
func Register(reg Registerer) {
	for v := range maps.Values(irBuiltins) {
		v(reg)
	}
}
