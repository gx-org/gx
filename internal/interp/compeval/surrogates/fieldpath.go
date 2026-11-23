// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package surrogates

import (
	"github.com/gx-org/gx/build/builder/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/src/gx/ast"
)

type fieldPath struct {
	core
	typ *ir.FieldPathType
}

var _ engine.FieldIndex = (*fieldPath)(nil)

func newFieldPath(path storepath.Path, typ ir.Type) (Element, error) {
	f := &fieldPath{core: core{path: path}}
	var err error
	f.typ, err = cast.To[*ir.FieldPathType](typ)
	return f, err
}

func (f *fieldPath) Type() ir.Type {
	return ir.MetaType()
}

func (f *fieldPath) FollowOn(x *ir.IndexExpr, _ ir.Element) (ir.Element, error) {
	return New(f.path, x.Typ)
}

func (f *fieldPath) Root() (*storepath.Proxy, error) {
	return nil, fmterr.Internalf("unexpected call to %T.Root", f)
}

func (f *fieldPath) Select(env *engine.Env, expr *ir.SelectorExpr) (ir.Element, error) {
	field, err := cast.To[*ir.FieldStorage](expr.Stor)
	if err != nil {
		return nil, err
	}
	selectPath := storepath.NewSelect(f.path, field.Field)
	return New(selectPath, field.Field.Type())
}

// castToArray returns the expression [axlengths(arg)...]DType(arg).
func (f *fieldPath) arrayType(x ir.Expr, dtype ir.Type) (ir.ArrayType, *ir.AxisExpr, error) {
	shape, err := builtins.CallAxLengths(x)
	if err != nil {
		return nil, nil, err
	}
	axisExpr := &ir.AxisExpr{X: &ir.UnpackExpr{
		X:      shape,
		EltTyp: ir.IntType(),
	}}
	rnk := &ir.Rank{
		Src: &ast.ArrayType{},
		Ax:  []ir.AxisLengths{axisExpr},
	}
	if !ir.CanBeNumber(dtype) {
		return nil, nil, fmterr.Internalf("cannot convert field index to array: %s is an invalid data type", dtype.ReferString(nil))
	}
	return ir.NewArrayType(x.Expr(), dtype, rnk), axisExpr, nil
}

func (f *fieldPath) buildPath(call *ir.FuncCallExpr, typ ir.Type) (storepath.Path, error) {
	return storepath.NewTypeIndex(&ir.IndexExpr{
		X:     call.Args[0],
		Index: call.Args[1],
		Typ:   typ,
	}, f.path), nil
}

func evalFieldIndexAxes(path storepath.Path, expr *ir.AxisExpr) (*elements.Slice, error) {
	return elements.NewSlice(ir.IntSliceType(), []ir.Element{
		elements.NewIRNodeWithType(expr.X, ir.IntSliceType(), path),
	})
}

func (f *fieldPath) ArrayCast(call *ir.FuncCallExpr, dtype ir.Type) (engine.NumericalElement, error) {
	arrayType, axisExpr, err := f.arrayType(call, dtype)
	if err != nil {
		return nil, err
	}
	path, err := f.buildPath(call, arrayType)
	if err != nil {
		return nil, err
	}
	return newArray(path, arrayType, func(ev ir.Evaluator, array ir.Type) (*elements.Slice, error) {
		return evalFieldIndexAxes(path, axisExpr)
	}), nil
}

func (f *fieldPath) Store() ir.Storage {
	return f.path.Store()
}
