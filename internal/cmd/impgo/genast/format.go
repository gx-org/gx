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

package genast

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"

	gxfmt "github.com/gx-org/gx/base/fmt"
)

func writeAndFormat(fset *token.FileSet, f *ast.File) (string, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		return "", err
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("cannot format generated source code: %v\nSource Code:\n%s", err, gxfmt.Number(string(buf.Bytes())))
	}
	return string(src), nil
}

func addLineBreaks(fset *token.FileSet, file *ast.File) {
	tf := fset.File(file.Pos())
	v := &lineBreaker{tf: tf}
	ast.Walk(v, file)
	lines := append([]int{}, tf.Lines()...)
	lines = append(lines, v.marks...)
	sort.Ints(lines)
	tf.SetLines(lines)
}

type lineBreaker struct {
	tf    *token.File
	marks []int
}

func (v *lineBreaker) mark(pos token.Pos) {
	v.marks = append(v.marks, v.tf.Offset(pos))
}

func (v *lineBreaker) Visit(n ast.Node) ast.Visitor {
	switch nT := n.(type) {
	case *ast.CompositeLit:
		v.mark(nT.Lbrace + 1)
		for _, el := range nT.Elts {
			v.mark(el.End())
		}
	case *ast.FuncDecl:
		v.mark(nT.End())
	}
	return v
}

// Write the AST into a buffer and returns the code as a string.
func (f *File) Write() (string, error) {
	file := *f.Src
	file.Decls = nil
	if len(f.imports) > 0 {
		file.Decls = []ast.Decl{&ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: f.imports,
		}}
	}
	file.Decls = append(file.Decls, f.Src.Decls...)
	src, err := writeAndFormat(token.NewFileSet(), &file)
	if err != nil {
		return "", err
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return "", err
	}
	addLineBreaks(fset, parsed)
	return writeAndFormat(fset, parsed)
}
