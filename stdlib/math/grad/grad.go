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

// Package grad implement GX functions to compute the gradient of GX functions.
package grad

import (
	"embed"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/stdlib/builtin"
)

//go:embed *.gx
var fs embed.FS

// Package description of the GX meta/grad package.
var Package = builtin.PackageBuilder{
	FullPath: "math/grad",
	Builders: []builtin.Builder{
		builtin.ParseSource(&fs),
		builtin.RegisterMacro("Func", FuncGrad),
	},
}

type gradMacro struct {
	fn  *ir.FuncDecl
	wrt string
}

// FuncGrad computes the gradient of a function.
func FuncGrad(call elements.CallAt, macro *cpevelements.Macro, args []elements.Element) (*cpevelements.SyntheticFunc, error) {
	fn, err := elements.FuncDeclFromElement(args[0])
	if err != nil {
		return nil, err
	}
	wrt, err := elements.StringFromElement(args[1])
	if err != nil {
		return nil, err
	}
	return cpevelements.NewSyntheticFunc(macro, &gradMacro{
		fn:  fn,
		wrt: wrt,
	}), nil
}

func (m *gradMacro) BuildType() (*ir.FuncType, error) {
	return m.fn.FType, nil
}

func (m *gradMacro) BuildBody(fetcher ir.Fetcher) (*ir.BlockStmt, bool) {
	return gradBlock(fetcher, m.fn.Body, m.wrt)
}

func gradBlock(fetcher ir.Fetcher, src *ir.BlockStmt, argName string) (*ir.BlockStmt, bool) {
	block := &ir.BlockStmt{List: make([]ir.Stmt, len(src.List))}
	for i, stmt := range src.List {
		var ok bool
		block.List[i], ok = gradStmt(fetcher, stmt, argName)
		if !ok {
			return nil, false
		}
	}
	return block, true
}

func gradStmt(fetcher ir.Fetcher, src ir.Stmt, argName string) (ir.Stmt, bool) {
	switch srcT := src.(type) {
	case *ir.ReturnStmt:
		return gradReturnStmt(fetcher, srcT, argName)
	default:
		return nil, fetcher.Err().Appendf(src.Source(), "gradient of %T statement not supported", srcT)
	}
}

func gradReturnStmt(fetcher ir.Fetcher, src *ir.ReturnStmt, argName string) (*ir.ReturnStmt, bool) {
	stmt := &ir.ReturnStmt{Results: make([]ir.Expr, len(src.Results))}
	for i, expr := range src.Results {
		res, ok := gradExpr(fetcher, expr, argName)
		if !ok {
			return nil, false
		}
		if res != nil {
			// The expression depends on arg: nothing left to do.
			stmt.Results[i] = res.x
			continue
		}
		// The expression does not depend on arg: replace it with a zero value.
		res, ok = zeroValueOf(fetcher, expr.Source(), expr.Type())
		if !ok {
			return nil, false
		}
		stmt.Results[i] = res.x
	}
	return stmt, true
}
