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
)

const assignDirectivePrefix = "//gx:="

type directivePrefix struct {
	dir        directive
	prefixOnly bool
}

var directives = map[string]directivePrefix{
	assignDirectivePrefix: directivePrefix{dir: assign, prefixOnly: false},
	"//gx:irmacro":        directivePrefix{dir: irmacro, prefixOnly: true},
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

func processScannerError(scope *scopeBlock, doc *ast.Comment, errScanner error) bool {
	errList, ok := errScanner.(scanner.ErrorList)
	if !ok {
		// Unknown error: we build a new error at the comment position
		// that includes the error type and its message.
		return scope.err().Appendf(doc, "%T:%s", errScanner, errScanner.Error())
	}
	for _, err := range errList {
		scope.err().Appendf(astErrorNode{doc: doc, err: err}, "%s", err.Msg)
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

func processFuncDirective(scope *scopeFile, fn *ast.FuncDecl) (directive, *ast.Comment, bool) {
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
			return dir, doc, scope.err().Appendf(doc, "a function can only have one GX directive")
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

func (f *funcDecl) processFuncAssignDirective(scope *scopeBlock, doc *ast.Comment) (*assignDirective, bool) {
	src := strings.TrimPrefix(doc.Text, assignDirectivePrefix)
	astExpr, err := parser.ParseExprFrom(scope.pkg().repr.FSet, f.bFile.name+":"+f.name().Name, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, processScannerError(scope, doc, err)
	}
	if f.meta != nil {
		return nil, scope.err().Appendf(doc, "a function can only have one GX directive")
	}
	expr, exprOk := processExpr(scope, astExpr)
	if !exprOk {
		return nil, false
	}
	call, callOk := expr.(*callExpr)
	if !callOk {
		return nil, scope.err().Appendf(doc, "GX equal directive (gx:=) only accept function call expression")
	}
	return &assignDirective{
		fun:  f,
		call: call,
	}, true
}

func fetchMacro(scope *scopeFile, call *callExpr) (ir.MacroImpl, bool) {
	selector, ok := call.src.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, scope.err().Appendf(call.src, "%T expression not supported: want <package name>.<function name>(...)", call.src.Fun)
	}
	packageIdent, ok := selector.X.(*ast.Ident)
	if !ok {
		return nil, scope.err().Appendf(call.src, "%T expression not supported: want <package name>.<function name>(...)", call.src.Fun)
	}
	pkgVal, ok := fetchType(scope, scope.namespace(), packageIdent)
	if !ok {
		return nil, false
	}
	pkgRef, ok := pkgVal.(*packageRef)
	if !ok {
		return nil, scope.err().Appendf(call.src, "%T.<function name> not supported: want <package name>.<function name>(...)", call.src.Fun)
	}
	pkg, err := scope.file().pkg.builder().Build(pkgRef.ext.Decl.Package.FullName())
	if err != nil {
		return nil, scope.err().Append(err)
	}
	funcName := selector.Sel.Name
	fun := pkg.IR().FindFunc(funcName)
	if fun == nil {
		return nil, scope.err().Appendf(call.src, "%s.%s undefined", packageIdent, funcName)
	}
	mFunc, ok := fun.(*ir.FuncMeta)
	if !ok {
		return nil, scope.err().Appendf(call.src, "cannot use %s.%s as a meta function", packageIdent, funcName)
	}
	macro, ok := mFunc.Impl.(ir.MacroImpl)
	if !ok {
		return nil, scope.err().Appendf(call.src, "cannot use %s.%s as a macro", packageIdent, funcName)
	}
	return macro, true
}

func (m *assignDirective) buildIR(parentScope *scopeFile) bool {
	args, _, ok := m.call.resolveArgs(parentScope)
	if !ok {
		return false
	}
	macro, ok := fetchMacro(parentScope, m.call)
	if !ok {
		return false
	}
	if err := macro(parentScope.evalFetcher(), &m.fun.ext, args); err != nil {
		parentScope.err().AppendAt(m.fun.ext.Src, err)
		return false
	}
	m.fun.funcType, ok = importFuncType(parentScope, m.fun.ext.FType)
	return ok
}
