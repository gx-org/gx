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

// Package genast provides utility to build an AST and save it to a file.
package genast

import (
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"strconv"
)

// Nil returns the nil identifier.
var Nil = Ident("nil").X

// Error returns the error identifier.
var Error = Ident("error").X

// File being generated.
type File struct {
	Src              *ast.File
	nextImport       int
	importPathToName map[string]IdentExpr

	imports []ast.Spec
}

// NewFile returns a new AST file helper.
func NewFile(name string) *File {
	return &File{
		Src: &ast.File{
			Name: Ident(name).X,
		},
		importPathToName: make(map[string]IdentExpr),
	}
}

// Import a package given its path and returns the alias to use.
func (f *File) Import(pkgPath string, useAlias bool) IdentExpr {
	alias := f.importPathToName[pkgPath]
	if alias != nil {
		return alias
	}
	var name *ast.Ident
	if useAlias {
		alias = Ident(fmt.Sprintf("pkg%d", f.nextImport))
		name = alias.X
		f.nextImport++
	} else {
		alias = Ident(path.Base(pkgPath))
	}
	f.imports = append(f.imports, &ast.ImportSpec{
		Name: name,
		Path: StringLit(pkgPath),
	})
	f.importPathToName[pkgPath] = alias
	return alias
}

// EmbedFile embed a file into the Go source code.
func (f *File) EmbedFile(name, filename string) IdentExpr {
	x := Ident(name)
	alias := f.Import("embed", false)
	f.Src.Decls = append(f.Src.Decls, &ast.GenDecl{
		Tok: token.VAR,
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{&ast.Comment{
				Text: fmt.Sprintf("//go:embed %s", filename),
			}},
		},
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{x.X},
				Type:  alias.Select("FS").X,
			},
		},
	})
	return x
}

// Var declares a variable in the file.
func (f *File) Var(name string, tp ast.Expr, value ast.Expr) IdentExpr {
	x := Ident(name)
	spec := &ast.ValueSpec{
		Names: []*ast.Ident{x.X},
		Type:  tp,
	}
	if value != nil {
		spec.Values = []ast.Expr{value}
	}
	f.Src.Decls = append(f.Src.Decls, &ast.GenDecl{
		Tok:   token.VAR,
		Specs: []ast.Spec{spec},
	})
	return x
}

// SliceLit returns a slice literal.
func SliceLit(tp ast.Expr, vals ...ast.Expr) *ast.CompositeLit {
	return &ast.CompositeLit{
		Type: &ast.ArrayType{Elt: tp},
		Elts: vals,
	}
}

// StructLit returns a slice literal.
func StructLit(tp ast.Expr, vals ...*ast.KeyValueExpr) Expr[*ast.CompositeLit] {
	exprs := make([]ast.Expr, len(vals))
	for i, val := range vals {
		exprs[i] = val
	}
	return Expr[*ast.CompositeLit]{
		X: &ast.CompositeLit{
			Type: tp,
			Elts: exprs,
		},
	}
}

// IntLit returns a integer literal.
func IntLit(i int) *ast.BasicLit {
	return &ast.BasicLit{
		Kind:  token.INT,
		Value: strconv.Itoa(i),
	}
}

// StringLit returns a string literal.
func StringLit(s string) *ast.BasicLit {
	return &ast.BasicLit{
		Kind:  token.STRING,
		Value: strconv.Quote(s),
	}
}

// Ident returns an identifier given a name.
func Ident(name string) IdentExpr {
	return &Expr[*ast.Ident]{X: &ast.Ident{Name: name}}
}
