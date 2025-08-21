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
)

type funcAttribute int

const (
	invalid funcAttribute = iota
	irmacro
	cpeval
	none
)

func (d funcAttribute) String() string {
	switch d {
	case none:
		return "none"
	case irmacro:
		return "irmacro"
	case cpeval:
		return "compeval"
	default:
		return "invalid"
	}
}

const funcAttributePrefix = "gx:"

var directives = map[string]funcAttribute{
	irmacro.String(): irmacro,
	cpeval.String():  cpeval,
	none.String():    none,
}

func trimCommentPrefix(cmt *ast.Comment) string {
	text := cmt.Text
	text = strings.TrimPrefix(text, "//")
	text = strings.TrimSpace(text)
	return text
}

func processFuncAttribute(pscope procScope, fn *ast.FuncDecl) (funcAttribute, *ast.Comment, bool) {
	if fn.Doc == nil {
		return none, nil, true
	}
	dir := none
	var comment *ast.Comment
	for _, doc := range fn.Doc.List {
		text := trimCommentPrefix(doc)
		if !strings.HasPrefix(text, funcAttributePrefix) {
			continue
		}
		if _, fnProc := funcProcessorFromDirective(text); fnProc != nil {
			continue
		}
		dirS := text[len(funcAttributePrefix):]
		docDir := directives[dirS]
		if docDir == invalid {
			return dir, doc, pscope.Err().Appendf(doc, "undefined directive %s", doc.Text)
		}
		if dir != none {
			return dir, doc, pscope.Err().Appendf(doc, "a function can only have one GX directive")
		}
		dir = docDir
		comment = doc
	}
	return dir, comment, true
}

type astErrorNode struct {
	doc *ast.Comment
	err *scanner.Error
}

var _ ast.Node = (*astErrorNode)(nil)

func (n astErrorNode) Pos() token.Pos {
	return n.doc.Pos() + token.Pos(n.err.Pos.Column) + token.Pos(len(assignPrefix)) - 1
}

func (n astErrorNode) End() token.Pos {
	return n.doc.End()
}

func processScannerError(pscope procScope, doc *ast.Comment, errScanner error) bool {
	errList, ok := errScanner.(scanner.ErrorList)
	if !ok {
		// Unknown error: we build a new error at the comment position
		// that includes the error type and its message.
		return pscope.Err().Appendf(doc, "%T:%s", errScanner, errScanner.Error())
	}
	for _, err := range errList {
		pscope.Err().Appendf(astErrorNode{doc: doc, err: err}, "%s", err.Msg)
	}
	return false
}

type funcProcessor func(pscope procScope, src *ast.FuncDecl, fn function, macroCall *callExpr) function

var funcProcessors = map[string]funcProcessor{
	assignPrefix:   processFuncAssignment,
	annotatePrefix: processFuncAnnotation,
}

func funcProcessorFromDirective(text string) (string, funcProcessor) {
	if len(text) <= 4 {
		return "", nil
	}
	prefix := text[:4]
	fnProc := funcProcessors[prefix]
	if fnProc == nil {
		return "", nil
	}
	return prefix, fnProc
}

func processFuncAnnotations(pscope procScope, src *ast.FuncDecl, fn function) (function, bool) {
	if src.Doc == nil {
		return fn, true
	}
	for _, doc := range src.Doc.List {
		text := trimCommentPrefix(doc)
		prefix, fnProc := funcProcessorFromDirective(text)
		if fnProc == nil {
			continue
		}
		text = strings.TrimPrefix(text, prefix)
		astExpr, err := parser.ParseExprFrom(pscope.pkgScope().pkg().fset, pscope.file().name+":"+src.Name.Name, text, parser.SkipObjectResolution)
		if err != nil {
			return nil, processScannerError(pscope, doc, err)
		}
		expr, ok := processExpr(pscope, astExpr)
		if !ok {
			return nil, false
		}
		macroCall, ok := expr.(*callExpr)
		if !ok {
			return nil, pscope.Err().Appendf(doc, "GX directive (%s) only accept function call expression", prefix)
		}
		fn = fnProc(pscope, src, fn, macroCall)
	}
	return fn, true
}
