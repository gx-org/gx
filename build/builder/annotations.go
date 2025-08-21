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

func (m *annotateFuncFromMacro) buildBody(fScope iFuncResolveScope, fn *irFunc) ([]*irFunc, bool) {
	aux, ok := m.function.buildBody(fScope, fn)
	if !ok {
		return nil, false
	}
	macroEl, ok := callMacroExpr(fScope.fileScope(), m.macroCall, fn.irFunc)
	if !ok {
		return nil, false
	}
	fnAnnotator, ok := macroEl.(cpevelements.FuncAnnotator)
	if !ok {
		return nil, fScope.Err().Appendf(m.macroCall.source(), "cannot use macro %s for function annotations", macroEl.Macro().Name())
	}
	return aux, fnAnnotator.Annotate(fScope, fn.irFunc)
}
