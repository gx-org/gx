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

package builder

import (
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"strings"

	"github.com/gx-org/gx/build/ir"
)

type directive int

const (
	none directive = iota
	assign
	irmacro
	cpeval
)

const assignDirectivePrefix = "//gx:="

type directivePrefix struct {
	dir        directive
	prefixOnly bool
}

var directives = map[string]directivePrefix{
	assignDirectivePrefix: directivePrefix{dir: assign, prefixOnly: false},
	"//gx:irmacro":        directivePrefix{dir: irmacro, prefixOnly: true},
	"//gx:compeval":       directivePrefix{dir: cpeval, prefixOnly: true},
}

type astErrorNode struct {
	doc *ast.Comment
	err *scanner.Error
}

var _ ast.Node = (*astErrorNode)(nil)

func (n astErrorNode) Pos() token.Pos {
	return n.doc.Pos() + token.Pos(n.err.Pos.Column) + token.Pos(len(assignDirectivePrefix)) - 1
}

func (n astErrorNode) End() token.Pos {
	return n.doc.End()
}

func processScannerError(pscope procScope, doc *ast.Comment, errScanner error) bool {
	errList, ok := errScanner.(scanner.ErrorList)
	if !ok {
		// Unknown error: we build a new error at the comment position
		// that includes the error type and its message.
		return pscope.err().Appendf(doc, "%T:%s", errScanner, errScanner.Error())
	}
	for _, err := range errList {
		pscope.err().Appendf(astErrorNode{doc: doc, err: err}, "%s", err.Msg)
	}
	return false
}

func directiveFromComment(doc string) directive {
	for prefix, dir := range directives {
		if !strings.HasPrefix(doc, prefix) {
			continue
		}
		if !dir.prefixOnly {
			return dir.dir
		}
		doc = strings.TrimSpace(doc)
		if prefix == doc {
			return dir.dir
		}
	}
	return none
}

func processFuncDirective(pscope procScope, fn *ast.FuncDecl) (directive, *ast.Comment, bool) {
	if fn.Doc == nil {
		return none, nil, true
	}
	var dir directive
	var comment *ast.Comment
	for _, doc := range fn.Doc.List {
		docDir := directiveFromComment(doc.Text)
		if docDir == none {
			continue
		}
		if dir != none {
			return dir, doc, pscope.err().Appendf(doc, "a function can only have one GX directive")
		}
		dir = docDir
		comment = doc
	}
	return dir, comment, true
}

type assignDirective struct {
	fun  *funcDecl
	call *callExpr
}

func (f *funcDecl) processFuncAssignDirective(pscope procScope, doc *ast.Comment) (*assignDirective, bool) {
	src := strings.TrimPrefix(doc.Text, assignDirectivePrefix)
	astExpr, err := parser.ParseExprFrom(pscope.pkgScope().pkg().fset, f.bFile.name+":"+f.name().Name, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, processScannerError(pscope, doc, err)
	}
	if f.meta != nil {
		return nil, pscope.err().Appendf(doc, "a function can only have one GX directive")
	}
	expr, exprOk := processExpr(pscope, astExpr)
	if !exprOk {
		return nil, false
	}
	call, callOk := expr.(*callExpr)
	if !callOk {
		return nil, pscope.err().Appendf(doc, "GX equal directive (gx:=) only accept function call expression")
	}
	return &assignDirective{
		fun:  f,
		call: call,
	}, true
}

func (m *assignDirective) buildExpr(scope *fileResolveScope) (*ir.FuncDecl, bool) {
	ext := &ir.FuncDecl{
		Src: m.fun.src,
	}
	args, ok := m.call.buildArgs(scope)
	if !ok {
		return ext, false
	}
	funcMetaExpr, ok := m.call.callee.buildExpr(scope)
	if !ok {
		return ext, false
	}
	funcMeta, ok := funcMetaExpr.(*ir.FuncMeta)
	if !ok {
		return ext, scope.err().Appendf(m.call.source(), "cannot use %s (%T) as a meta function", funcMeta.Name(), funcMetaExpr)
	}
	if funcMeta.Impl == nil {
		return ext, scope.err().Appendf(m.call.source(), "meta function %s has no implementation", funcMeta.Name())
	}
	compEval, compEvalOk := scope.compEval()
	if !compEvalOk {
		return ext, false
	}
	ext, ok = funcMeta.Impl(compEval, ext, args)
	return ext, ok
}
