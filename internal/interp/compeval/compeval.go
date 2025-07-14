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

// Package compeval runs GX code at compile time.
package compeval

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp"
)

// EvalExpr evaluates a GX expression into an interpreter element.
func EvalExpr(ctx evaluator.Context, expr ir.Expr) (cpevelements.Element, error) {
	val, err := ctx.EvalExpr(expr)
	if err != nil {
		return nil, err
	}
	el, ok := val.(cpevelements.Element)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", val, reflect.TypeFor[cpevelements.Element]().String())
	}
	return el, nil
}

// NewOptionVariable creates a package option to set a static variable of a package with its corresponding symbolic element.
func NewOptionVariable(vr *ir.VarExpr) options.PackageOption {
	src := elements.NewNodeAt[ir.Storage](vr.Decl.FFile, vr)
	return elements.PackageVarSetElement{
		Pkg:   vr.Decl.FFile.Package.FullName(),
		Var:   vr.VName.Name,
		Value: cpevelements.NewVariable(src),
	}
}

// EvalInt evaluates an expression to return an int.
func EvalInt(fetcher ir.Fetcher, expr ir.Expr) (int, error) {
	el, err := fetcher.EvalExpr(expr)
	if err != nil {
		return 0, err
	}
	val := canonical.ToValue(el)
	if val == nil {
		return 0, fmterr.Errorf(fetcher.File().FileSet(), expr.Source(), "expected axis literals, but expression %s cannot be evaluated at compile time", expr.String())
	}
	if !val.IsInt() {
		return 0, fmterr.Errorf(fetcher.File().FileSet(), expr.Source(), "cannot use %s as static int value in axis specification", val.String())
	}
	valInt, _ := val.Int64()
	return int(valInt), nil
}

// EvalRank evaluates an expression to build the rank of an array.
func EvalRank(fetcher ir.Fetcher, expr ir.Expr) (ir.ArrayRank, []canonical.Canonical, error) {
	rankVal, err := fetcher.EvalExpr(expr)
	if err != nil {
		return nil, nil, err
	}
	slice, ok := rankVal.(*interp.Slice)
	if !ok {
		return nil, nil, fmterr.Internalf(fetcher.File().FileSet(), expr.Source(), "cannot build a rank from %s (%T): not supported", expr.String(), rankVal)
	}
	axes := make([]ir.AxisLengths, slice.Len())
	cans := make([]canonical.Canonical, slice.Len())
	for i, el := range slice.Elements() {
		ex, ok := el.(ir.Canonical)
		if !ok {
			return nil, nil, fmterr.Internalf(fetcher.File().FileSet(), expr.Source(), "cannot build an axis expression from element %T: not supported", el)
		}
		irExpr, err := ex.Expr()
		if err != nil {
			return nil, nil, err
		}
		axes[i] = &ir.AxisExpr{
			X: irExpr,
		}
		cans[i] = el.(canonical.Canonical)
	}
	return &ir.Rank{Ax: axes}, cans, nil
}
