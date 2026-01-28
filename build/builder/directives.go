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
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
)

type funcAttribute int

const (
	invalid funcAttribute = iota
	irmacro
	funcAnnotator
	fieldAnnotator
	cpeval
	none
)

func (d funcAttribute) String() string {
	switch d {
	case none:
		return "none"
	case irmacro:
		return "irmacro"
	case funcAnnotator:
		return "funcAnnotator"
	case fieldAnnotator:
		return "fieldAnnotator"
	case cpeval:
		return "compeval"
	default:
		return "invalid"
	}
}

const directivePrefix = "gx:"

var directives = map[string]funcAttribute{
	irmacro.String():        irmacro,
	funcAnnotator.String():  funcAnnotator,
	fieldAnnotator.String(): fieldAnnotator,
	cpeval.String():         cpeval,
	none.String():           none,
}

type directive struct {
	prefix string
	code   string
}

func newDirective(text string) directive {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, directivePrefix) {
		return directive{}
	}
	return directive{
		prefix: directivePrefix,
		code:   text[len(directivePrefix):],
	}
}

func (d directive) empty() bool {
	return d.code == ""
}

func (d directive) addPrefix(prefix string) directive {
	if !strings.HasPrefix(d.code, prefix) {
		return directive{}
	}
	return directive{
		prefix: d.prefix + prefix,
		code:   d.code[len(prefix):],
	}
}

func (d directive) addPrefixLen(l int) directive {
	if len(d.code) < l {
		return directive{}
	}
	return directive{
		prefix: d.prefix + d.code[:l],
		code:   d.code[l:],
	}
}

func (d directive) process(pscope procScope, src ast.Node) (*callExpr, bool) {
	astExpr, err := parser.ParseExprFrom(pscope.pkg().fset, pscope.file().name, d.code, parser.SkipObjectResolution)
	if err != nil {
		return nil, processScannerError(pscope.Err(), src, err)
	}
	expr, ok := processExpr(pscope, astExpr)
	if !ok {
		return nil, false
	}
	macroCall, ok := expr.(*callExpr)
	if !ok {
		return nil, pscope.Err().Appendf(src, "GX directive (%s) only accept function call expression", d.prefix)
	}
	return macroCall, ok
}

func trimCommentPrefix(cmt *ast.Comment) directive {
	text := cmt.Text
	text = strings.TrimPrefix(text, "//")
	return newDirective(text)
}

func processFuncAttribute(pscope procScope, fn *ast.FuncDecl) (funcAttribute, *ast.Comment, bool) {
	if fn.Doc == nil {
		return none, nil, true
	}
	dir := none
	var comment *ast.Comment
	for _, doc := range fn.Doc.List {
		genericDir := trimCommentPrefix(doc)
		if genericDir.empty() {
			continue
		}
		if _, fnProc := funcProcessorFromDirective(genericDir); fnProc != nil {
			continue
		}
		docDir := directives[genericDir.code]
		if docDir == invalid {
			return dir, doc, pscope.Err().Appendf(doc, "undefined directive %s", doc.Text)
		}
		if dir != none {
			return dir, doc, pscope.Err().Appendf(doc, "directive %s incompatible with previous directive %s%s", genericDir, directivePrefix, dir)
		}
		dir = docDir
		comment = doc
	}
	return dir, comment, true
}

func processScannerError(appender *fmterr.Appender, src ast.Node, errScanner error) bool {
	errList, ok := errScanner.(scanner.ErrorList)
	if !ok {
		// Unknown error: we build a new error at the comment position
		// that includes the error type and its message.
		return appender.Appendf(src, "%T:%s", errScanner, errScanner.Error())
	}
	for _, err := range errList {
		pos := fmterr.Pos{Begin: err.Pos}
		if src != nil {
			pos = appender.FSet().Pos(src)
		}
		appender.Append(pos.Error(errors.New(err.Msg)))
	}
	return false
}

type funcProcessor func(pscope procScope, src *ast.FuncDecl, fn function, macroCall *callExpr) function

var funcProcessors = map[string]funcProcessor{
	assignPrefix:   processFuncAssignment,
	annotatePrefix: processFuncAnnotation,
}

func funcProcessorFromDirective(dir directive) (directive, funcProcessor) {
	nextDir := dir.addPrefixLen(1)
	if nextDir.empty() {
		return directive{}, nil
	}
	fnProc := funcProcessors[nextDir.prefix]
	if fnProc == nil {
		return directive{}, nil
	}
	return nextDir, fnProc
}

func processFuncAnnotations(pscope procScope, src *ast.FuncDecl, fn function) (function, bool) {
	if src.Doc == nil {
		return fn, true
	}
	for _, doc := range src.Doc.List {
		dir := trimCommentPrefix(doc)
		dir, fnProc := funcProcessorFromDirective(dir)
		if fnProc == nil {
			continue
		}
		macroCall, ok := dir.process(pscope, doc)
		if !ok {
			return nil, false
		}
		fn = fnProc(pscope, src, fn, macroCall)
	}
	return fn, true
}
