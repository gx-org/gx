// Copyright 2026 Google LLC
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

package gengo

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/gx-org/gx/internal/cmd/impgo/genast"
)

type directives struct {
	g  goGenerator
	xs []ast.Expr
}

func (g goGenerator) newDirectives(xs ...ast.Expr) *directives {
	return &directives{g: g, xs: xs}
}

type funcBody struct {
	dir  *directives
	f    *types.Func
	env  genast.IdentExpr
	call genast.IdentExpr
	recv genast.IdentExpr
	args genast.IdentExpr
}

func (d *directives) newBody(f *types.Func) *funcBody {
	return &funcBody{
		dir:  d,
		f:    f,
		env:  genast.Ident("env"),
		call: genast.Ident("call"),
		recv: genast.Ident("recv"),
		args: genast.Ident("args"),
	}
}

func (fb *funcBody) convertArg(block *genast.Block, i int, tp *types.Var) genast.IdentExpr {
	return block.Assign(fmt.Sprintf("arg%d", i), fb.args.IndexAt(i).X)
}

func (fb *funcBody) gxFromGoBasic(tp *types.Basic, name genast.IdentExpr) *ast.CallExpr {
	switch tp.Kind() {
	case types.Float32:
		return genast.CallExpr(
			fb.dir.g.values.Select("AtomFloatValue").Index(genast.Ident("float32").X).X,
			genast.CallExpr(fb.dir.g.ir.Select("Float32Type").X),
			name.X,
		)
	}
	return genast.CallExpr(
		genast.Ident("unknown").X,
		name.X,
	)
}

func (fb *funcBody) gxFromGo(block *genast.Block, vr *types.Var, name genast.IdentExpr) genast.IdentExpr {
	var call *ast.CallExpr
	switch typT := vr.Type().(type) {
	case *types.Basic:
		call = fb.gxFromGoBasic(typT, name)
	}
	gxVal := genast.Ident(name.X.Name + "GX")
	err := genast.Ident("err")
	block.UnpackAssign([]genast.IdentExpr{gxVal, err}, call)
	block.Add(&ast.IfStmt{
		Cond: &ast.BinaryExpr{
			Op: token.NEQ,
			X:  err.X,
			Y:  genast.Nil,
		},
		Body: genast.NewBlock().Return(
			genast.Nil,
			err.X,
		).Block(),
	})
	return gxVal
}

func (fb *funcBody) toGXResults(block *genast.Block, results *types.Tuple, vars []genast.IdentExpr) []genast.IdentExpr {
	names := make([]genast.IdentExpr, results.Len())
	for i := range results.Len() {
		names[i] = fb.gxFromGo(block, results.At(i), vars[i])
	}
	return names
}

func (fb *funcBody) statements() *genast.Block {
	block := &genast.Block{}
	sig := fb.f.Signature()
	params := sig.Params()
	// Collect all arguments.
	args := make([]genast.IdentExpr, params.Len())
	for i := range params.Len() {
		args[i] = fb.convertArg(block, i, params.At(i))
	}
	// Prepare the results.
	results := sig.Results()
	res := make([]genast.IdentExpr, results.Len())
	for i := range results.Len() {
		res[i] = genast.Ident(fmt.Sprintf("res%d", i))
	}
	// Call the function.
	block.UnpackAssign(res, genast.CallExpr(
		fb.dir.g.gopkg.Select(fb.f.Name()).X,
		genast.AstExprs(args)...,
	))
	gxResults := fb.toGXResults(block, results, res)
	// Convert the results from Go to GX.
	// Return the results.
	block.Return(
		genast.SliceLit(
			fb.dir.g.ir.Select("Element").X,
			genast.AstExprs(gxResults)...,
		),
		genast.Nil,
	)
	return block
}

func (d *directives) buildFuncImpl(f *types.Func) (*ast.Ident, error) {
	body := d.newBody(f)
	decl := d.g.file.FuncDecl(
		"eval"+f.Name(),
		&ast.FuncType{
			Params: genast.Fields(
				genast.Field(body.env, d.g.engine.Select("Env").X),
				genast.Field(body.call, d.g.ir.Select("FuncCallExpr").Star().X),
				genast.Field(body.recv, d.g.ir.Select("Element").X),
				genast.Field(body.args, &ast.ArrayType{
					Elt: d.g.ir.Select("Element").X,
				}),
			),
			Results: genast.Fields(
				genast.Field(nil, &ast.ArrayType{
					Elt: d.g.ir.Select("Element").X,
				}),
				genast.Field(nil, genast.Error),
			),
		},
		body.statements())
	return decl.Name, nil
}

func (d *directives) Func(f *types.Func) error {
	funImpl, err := d.buildFuncImpl(f)
	if err != nil {
		return err
	}
	d.xs = append(d.xs, genast.CallExpr(
		d.g.builtin.Select("ImplementBuiltin").X,
		genast.StringLit(f.Name()),
		funImpl,
	))
	return nil
}

func (d *directives) Struct(name *types.TypeName, s *types.Struct) error {
	return nil
}

func (d *directives) Interface(name *types.TypeName, s *types.Interface) error {
	return nil
}
