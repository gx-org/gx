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

package grad

import (
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir/annotations"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
)

type (
	setAnnotation struct {
		partials []ir.PkgFunc
	}

	setAnnotationMacro struct {
		cpevelements.CoreMacroElement
		grad ir.PkgFunc

		paramName string
	}
)

var _ ir.FuncAnnotator = (*setAnnotationMacro)(nil)

func setMacro(mac *ir.Macro) *ir.Macro {
	return mac.File().Package.FindFunc("SetFor").(*ir.Macro)
}

// SetGrad sets the gradient of a function.
func SetGrad(file *ir.File, call *ir.CallExpr, macro *ir.Macro, args []ir.Element) (ir.MacroElement, error) {
	grad, err := interp.PkgFuncFromElement(args[0])
	if err != nil {
		return nil, errors.Errorf("%s is not a function", args[0].Type().String())
	}

	return newSetMacro(file, call, macro, grad, "")
}

// SetGradFor sets the gradient of a function.
func SetGradFor(file *ir.File, call *ir.CallExpr, macro *ir.Macro, args []ir.Element) (ir.MacroElement, error) {
	grad, err := interp.PkgFuncFromElement(args[0])
	if err != nil {
		return nil, errors.Errorf("%s is not a function", args[1].Type().String())
	}
	fieldName, err := elements.StringFromElement(args[1])
	if err != nil {
		return nil, err
	}
	return newSetMacro(file, call, macro, grad, fieldName)
}

func newSetMacro(file *ir.File, call *ir.CallExpr, mac *ir.Macro, grad ir.PkgFunc, paramName string) (ir.MacroElement, error) {
	return &setAnnotationMacro{
		CoreMacroElement: cpevelements.MacroElementWithKey(mac, file, call, setMacro(mac)),
		grad:             grad,
		paramName:        paramName,
	}, nil
}

func findNameInFields(paramName string, fields *ir.FieldList) int {
	for i, param := range fields.Fields() {
		if param.Name == nil {
			continue
		}
		if param.Name.Name == paramName {
			return i
		}
	}
	return -1
}

func (m *setAnnotationMacro) Annotate(fetcher ir.Fetcher, fn ir.PkgFunc) bool {
	fType := fn.FuncType()
	paramToFunc := annotations.GetDef(fn, m.Key(), func() *setAnnotation {
		return &setAnnotation{
			partials: make([]ir.PkgFunc, fType.Params.Len()),
		}
	})
	paramPos := -1
	if m.paramName == "" {
		if fn.FuncType().Params.Len() != 1 {
			return fetcher.Err().Appendf(m.Source(), "cannot set gradient of %s: requires a single argument", fn.Name())
		}
		paramPos = 0
	} else {
		paramPos = findNameInFields(m.paramName, fType.Params)
	}
	if paramPos < 0 {
		return fetcher.Err().Appendf(m.Source(), "function %s has no parameter %s", fn.Name(), m.paramName)
	}
	prev := paramToFunc.partials[paramPos]
	if prev != nil {
		return fetcher.Err().Appendf(m.Source(), "gradient for parameter %s has already been set", m.paramName)
	}
	paramToFunc.partials[paramPos] = m.grad
	return true
}

func gradFromAnnotation(fetcher ir.Fetcher, src ir.Func, paramToFunc *setAnnotation, wrt string) (*ast.Ident, bool) {
	paramPos := findNameInFields(wrt, src.FuncType().Params)
	if paramPos < 0 {
		return nil, fetcher.Err().Appendf(src.Source(), "function %s has no parameter %s", src.ShortString(), wrt)
	}
	pkgFunc := paramToFunc.partials[paramPos]
	if pkgFunc == nil {
		return nil, fetcher.Err().Appendf(src.Source(), "no gradient defined for parameter %s of function %s", wrt, src.ShortString())
	}
	return &ast.Ident{Name: pkgFunc.Name()}, true
}
