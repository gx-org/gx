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
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
)

// SetGrad sets the gradient of a function.
func SetGrad(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (*cpevelements.SyntheticFunc, error) {
	fn, ok := args[0].(ir.PkgFunc)
	if !ok {
		return nil, errors.Errorf("%T not an IR function", args[0])
	}
	grad, err := interp.PkgFuncFromElement(args[1])
	if err != nil {
		return nil, errors.Errorf("%T is not a function element", args[1])
	}
	return cpevelements.NewSyntheticFunc(&setAnnotation{
		macro: macro,
		fn:    fn,
		grad:  grad,
	}), nil
}

type setAnnotation struct {
	macro *cpevelements.Macro
	fn    ir.PkgFunc
	grad  ir.PkgFunc
}

const setKey = "set"

func (m *setAnnotation) BuildIR(errApp fmterr.ErrAppender, src *ast.FuncDecl, file *ir.File, fType *ir.FuncType) (ir.PkgFunc, bool) {
	ext := m.fn.New(src, file, fType)
	ext.Annotations().Append(
		m.macro.Func().File().Package,
		setKey,
		m.grad,
	)
	return ext, true
}

func findSetAnnotation(stor ir.Storage) *ir.Annotation {
	pkgFunc, ok := stor.(ir.PkgFunc)
	if !ok {
		return nil
	}
	for _, ann := range pkgFunc.Annotations().Anns {
		if ann.Key() == "math/grad:"+setKey {
			return ann
		}
	}
	return nil
}

func gradFuncWithSet(fetcher ir.Fetcher, src *ir.FuncValExpr, ann *ir.Annotation) (ast.Expr, bool) {
	pkgFunc, ok := ann.Value().(ir.PkgFunc)
	if !ok {
		return nil, fetcher.Err().Appendf(src.Source(), "invalid annotation value: got %T but want a package function", ann.Value())
	}
	return &ast.Ident{Name: pkgFunc.Name()}, true
}
