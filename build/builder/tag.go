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
	"strconv"
	"strings"

	"github.com/gx-org/gx/base/ordered"
	"github.com/gx-org/gx/build/ir"
)

type tag struct {
	src  *ast.BasicLit
	tags []directive
}

func processTag(pscope typeProcScope, lit *ast.BasicLit) (*tag, bool) {
	t := &tag{src: lit}
	if t.src == nil {
		return t, true
	}
	tag, err := strconv.Unquote(t.src.Value)
	if err != nil {
		return t, pscope.Err().AppendAt(lit, err)
	}
	tags := strings.Split(tag, ";")
	for _, tag := range tags {
		dir := newDirective(tag)
		if dir.empty() {
			continue
		}
		t.tags = append(t.tags, dir)
	}
	return t, true
}

func buildTags(dscope *defineLocalScope, list *fieldList, irlist *ir.FieldList) bool {
	ok := true
	compEval, ok := dscope.compEval()
	if !ok {
		return false
	}
	checkers := ordered.NewMap[ir.FieldListCheckImpl, bool]()
	for i, group := range list.list {
		if !group.tag.build(dscope, compEval, checkers, irlist.List[i]) {
			ok = false
		}
	}
	for checker := range checkers.Keys() {
		ok = checker.Check(compEval, irlist) && ok
	}
	return ok
}

func (t *tag) buildDirective(dscope *defineLocalScope, compEval *compileEvaluator, grp *ir.FieldGroup, dir directive) (ir.FieldListCheckImpl, bool) {
	macroCall, ok := dir.process(dscope.fileScope().procScope(), t.src)
	if !ok {
		return nil, false
	}
	call, annotator, ok := evalFieldAnnotator(dscope, compEval, macroCall)
	if !ok {
		return nil, false
	}
	args, ok := evalCallArgs(dscope, compEval, call)
	if !ok {
		return nil, false
	}
	return annotator.Annotate(compEval, grp, call, args)
}

func (t *tag) build(dscope *defineLocalScope, compEval *compileEvaluator, checkers *ordered.Map[ir.FieldListCheckImpl, bool], grp *ir.FieldGroup) bool {
	ok := true
	for _, dir := range t.tags {
		annotationDir := dir.addPrefix("@")
		if annotationDir.empty() {
			ok = dscope.Err().Appendf(t.src, "expect annotation gx:@... but found gx:%s", dir.code)
			continue
		}
		checker, dirOk := t.buildDirective(dscope, compEval, grp, annotationDir)
		if !dirOk {
			ok = false
			continue
		}
		if checker != nil {
			checkers.Store(checker, true)
		}
	}
	return ok
}
