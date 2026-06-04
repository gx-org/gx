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

// Package errors provides errors GX functions.
package errors

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/stdlib/builtin"
)

// Package description of the GX fmt package.
var Package = builtin.PackageBuilder{
	FullPath: "errors",
	Builders: []builtin.Builder{
		builtin.ParseSource(),
	},
}

// Errorf returns a GX error.
func Errorf(env engine.Env, format string, a ...any) (ir.Element, error) {
	pkg, err := env.Engine().Importer().Import("errors")
	if err != nil {
		return nil, err
	}
	fn := pkg.FindFunc("New")
	eval, err := env.ExprEval().Sub(pkg.Files["errors/errors.gx"], nil)
	if err != nil {
		return nil, err
	}
	return eval.EvalExpr(&ir.FuncCallExpr{
		Callee: ir.NewFuncValExpr(
			&ir.Ident{Src: &ast.Ident{Name: "New"}, Stor: fn},
			fn,
		),
		Args: []ir.Expr{&ir.StringLiteral{
			Src: &ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(fmt.Sprintf(format, a...)),
			},
		}},
	})
}
