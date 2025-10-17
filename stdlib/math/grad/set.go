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

	"github.com/gx-org/gx/build/ir/annotations"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
)

type setAnnotation struct {
	partials []ir.PkgFunc
}

var setKey = annotations.NewKey(setAnnotation{})

// SetGrad sets the gradient of a function.
func SetGrad(fetcher ir.Fetcher, ann *ir.Annotator, fn ir.PkgFunc, call *ir.CallExpr, args []ir.Element) bool {
	params := fn.FuncType().Params.Fields()
	switch len(params) {
	case 0:
		return fetcher.Err().Appendf(call.Source(), "cannot set gradient of %s: function has no argument", fn.Name())
	case 1:
	default:
		return fetcher.Err().Appendf(call.Source(), "use %s.SetFor to set the gradient of a function with more than one parameter", fn.File().Package.Name.Name)
	}
	field := params[0]
	paramName := "_"
	if field.Name != nil {
		paramName = field.Name.Name
	}
	return annotate(fetcher, ann, fn, call, paramName, 0, args[0])
}

// SetGradFor sets the gradient of a function.
func SetGradFor(fetcher ir.Fetcher, ann *ir.Annotator, fn ir.PkgFunc, call *ir.CallExpr, args []ir.Element) bool {
	arg0, err := fetcher.EvalExpr(call.Args[0])
	if err != nil {
		return fetcher.Err().AppendAt(call.Args[0].Source(), err)
	}
	paramName, err := elements.StringFromElement(arg0)
	if err != nil {
		return fetcher.Err().AppendAt(call.Args[0].Source(), err)
	}
	paramPos := findNameInFields(paramName, fn.FuncType().Params)
	if paramPos < 0 {
		return fetcher.Err().Appendf(call.Args[0].Source(), "function %s has no parameter %s", fn.Name(), paramName)
	}
	return annotate(fetcher, ann, fn, call, paramName, paramPos, args[1])
}

func annotate(fetcher ir.Fetcher, ann *ir.Annotator, fn ir.PkgFunc, call *ir.CallExpr, paramName string, paramPos int, gradEl ir.Element) bool {
	fType := fn.FuncType()
	paramToFunc := annotations.GetDef(fn, setKey, func() *setAnnotation {
		return &setAnnotation{
			partials: make([]ir.PkgFunc, fType.Params.Len()),
		}
	})
	prev := paramToFunc.partials[paramPos]
	if prev != nil {
		return fetcher.Err().Appendf(call.Source(), "gradient for parameter %s has already been set", paramName)
	}
	gradFn, err := interp.PkgFuncFromElement(gradEl)
	if err != nil {
		return fetcher.Err().AppendAt(call.Source(), err)
	}
	paramToFunc.partials[paramPos] = gradFn
	return true
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
