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

const assignPrefix = "gx:="

type assignFuncFromMacro struct {
	coreSyntheticFunc
	fn        function
	macroCall *callExpr
}

func processFuncAssignment(pscope procScope, src *ast.FuncDecl, fn function, macroCall *callExpr) function {
	return &assignFuncFromMacro{
		fn: fn,
		coreSyntheticFunc: coreSyntheticFunc{
			bFile: pscope.file(),
			src:   src,
		},
		macroCall: macroCall,
	}
}

func (f *assignFuncFromMacro) buildSignature(pkgScope *pkgResolveScope) (ir.Func, iFuncResolveScope, bool) {
	fScope, ok := pkgScope.newFileScope(f.bFile)
	if !ok {
		return nil, nil, false
	}
	// Build the signature of the underlying function.
	underFun, _, ok := f.fn.buildSignature(pkgScope)
	if !ok {
		return nil, nil, false
	}
	underPkgFun, ok := underFun.(ir.PkgFunc)
	if !ok {
		return nil, nil, fScope.Err().AppendInternalf(f.fn.source(), "%T not a package function", underFun)
	}
	macroEl, ok := callMacroExpr(fScope, f.macroCall, underPkgFun)
	if !ok {
		return nil, nil, false
	}
	// Return the result as a synthetic function.
	fnBuilder, ok := macroEl.(cpevelements.FuncASTBuilder)
	if !ok {
		return nil, nil, fScope.Err().Appendf(f.fn.source(), "cannot use macro %s for function assignment", macroEl.Macro().Name())
	}
	synthFunc, synthScope, ok := (&syntheticFunc{
		coreSyntheticFunc: f.coreSyntheticFunc,
		fnBuilder:         fnBuilder,
		underFun:          underFun,
	}).buildSignatureFScope(fScope)
	if !ok {
		return nil, nil, false
	}
	return synthFunc, synthScope, true
}
