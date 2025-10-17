// Copyright 2025 Google LLC
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

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
)

const annotatePrefix = "gx:@"

type annotateFuncFromMacro struct {
	function
	macroCall *callExpr
}

func processFuncAnnotation(pscope procScope, src *ast.FuncDecl, fn function, macroCall *callExpr) function {
	return &annotateFuncFromMacro{
		function:  fn,
		macroCall: macroCall,
	}
}

func (m *annotateFuncFromMacro) buildAnnotations(fnScope iFuncResolveScope, fn *irFunc) bool {
	ok := m.function.buildAnnotations(fnScope, fn)
	if !ok {
		return false
	}
	compEval, ok := fnScope.compEval()
	if !ok {
		return false
	}
	call, annotator, ok := evalMetaCallee[*cpevelements.Annotator](fnScope, compEval, m.macroCall)
	if !ok {
		return false
	}
	els := make([]ir.Element, len(call.Args))
	for i, arg := range call.Args {
		var err error
		els[i], err = compEval.EvalExpr(arg)
		if err != nil {
			fnScope.Err().AppendAt(arg.Source(), err)
			ok = false
		}
	}
	if !ok {
		return false
	}
	return annotator.Annotate(compEval, fn.irFunc, call, els)
}
