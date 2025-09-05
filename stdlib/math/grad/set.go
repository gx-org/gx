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
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
)

type (
	setAnnotation = ir.AnnotationT[map[string]ir.PkgFunc]

	setAnnotationMacro struct {
		cpevelements.CoreMacroElement
		fn    ir.PkgFunc
		grad  ir.PkgFunc
		param *ir.Field
	}
)

var _ cpevelements.FuncAnnotator = (*setAnnotationMacro)(nil)

// SetGrad sets the gradient of a function.
func SetGrad(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (cpevelements.MacroElement, error) {
	fn, ok := args[0].(ir.PkgFunc)
	if !ok {
		return nil, errors.Errorf("%T not an IR function", args[0])
	}
	if fn.FuncType().Params.Len() != 1 {
		return nil, errors.Errorf("cannot set gradient of %s: requires a single argument", fn.Name())
	}
	params := fn.FuncType().Params.Fields()
	if len(params) != 1 {
		return nil, errors.Errorf("cannot set gradient of %s: requires a single argument", fn.Name())
	}
	grad, err := interp.PkgFuncFromElement(args[1])
	if err != nil {
		return nil, errors.Errorf("%s is not a function", args[1].Type().String())
	}

	return newSetMacro(fn, macro, grad, params[0])
}

// SetGradFor sets the gradient of a function.
func SetGradFor(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (cpevelements.MacroElement, error) {
	fn, ok := args[0].(ir.PkgFunc)
	if !ok {
		return nil, errors.Errorf("%T not an IR function", args[0])
	}
	grad, err := interp.PkgFuncFromElement(args[1])
	if err != nil {
		return nil, errors.Errorf("%s is not a function", args[1].Type().String())
	}
	fieldName, err := elements.StringFromElement(args[2])
	if err != nil {
		return nil, err
	}
	param := fn.FuncType().Params.FindField(fieldName)
	if param == nil {
		return nil, errors.Errorf("function %s has no parameter %s", fn.Name(), fieldName)
	}
	return newSetMacro(fn, macro, grad, param)
}

func newSetMacro(fn ir.PkgFunc, macro *cpevelements.Macro, grad ir.PkgFunc, param *ir.Field) (cpevelements.MacroElement, error) {
	return &setAnnotationMacro{
		CoreMacroElement: cpevelements.CoreMacroElement{Mac: macro},
		fn:               fn,
		grad:             grad,
		param:            param,
	}, nil
}

const setKey = "math/grad:set"

func (m *setAnnotationMacro) Annotate(fetcher ir.Fetcher, fn ir.PkgFunc) bool {
	ann := ir.AnnotationFrom[map[string]ir.PkgFunc](fn, setKey)
	if ann == nil {
		ann = ir.NewAnnotation[map[string]ir.PkgFunc](setKey, make(map[string]ir.PkgFunc))
		fn.Annotations().Set(ann)
	}
	paramToFunc := ann.Value()
	paramName := m.param.Name.Name
	prev := paramToFunc[paramName]
	if prev != nil {
		return fetcher.Err().Appendf(m.fn.Source(), "gradient for parameter %s has already been set", paramName)
	}
	paramToFunc[paramName] = m.grad
	return true
}

func findSetAnnotation(stor ir.Storage) *setAnnotation {
	pkgFunc, ok := stor.(ir.PkgFunc)
	if !ok {
		return nil
	}
	return ir.AnnotationFrom[map[string]ir.PkgFunc](pkgFunc, setKey)
}

func gradFromAnnotation(fetcher ir.Fetcher, src ir.Func, ann *setAnnotation, wrt string) (ast.Expr, bool) {
	paramToFunc := ann.Value()
	pkgFunc := paramToFunc[wrt]
	if pkgFunc == nil {
		return nil, fetcher.Err().Appendf(src.Source(), "no gradient defined for parameter %s of function %s", wrt, src.Name())
	}
	return &ast.Ident{Name: pkgFunc.Name()}, true
}
