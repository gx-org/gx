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

package ir

import (
	"fmt"
	"go/ast"

	"github.com/gx-org/gx/build/ir/irkind"
)

const (
	// DefineAxisGroup is the prefix used in parameters to define an axis group.
	DefineAxisGroup = "___"
	// DefineAxisLength is the prefix used in parameters to define an axis length.
	DefineAxisLength = "_"
)

type (
	// AxisLengths specification of an array.
	AxisLengths interface {
		Node
		StringSourcer

		axExprString(*File) string

		// Type of the axis.
		Type() Type

		// AsExpr returns the axis value as an expression.
		// Can return nil if the axis has not been resolved.
		AsExpr() Expr

		// Equal returns true if two axis are equal.
		Equal(TypeCmp, AxisLengths) (bool, error)

		// Specialise the axis length given a context.
		Specialise(Specialiser) ([]AxisLengths, bool)

		// UnifyWith unifies axis lengths with a given target.
		UnifyWith(Unifier, []AxisLengths) ([]AxisLengths, bool)

		// Instantiate the rank into another rank.
		Instantiate(Fetcher, Specialiser) ([]AxisLengths, bool)

		// IndexForVarArgs returns an axis specific to a given index in varargs.
		IndexForVarArgs(int) AxisLengths
	}

	// ExprUnpacker is an expression that can unpack into multiple expressions.
	// Implemented by a slice literal.
	ExprUnpacker interface {
		Expr
		Unpack() []Expr
	}

	// AxisExpr is an array axis specified using an expression.
	AxisExpr struct {
		// X computes the size of the axis.
		X Expr
	}
)

var _ AxisLengths = (*AxisExpr)(nil)

func (*AxisExpr) node() {}

// Node returns the source expression specifying the axis length.
func (dm *AxisExpr) Node() ast.Node { return dm.Expr() }

// Expr returns the AST of the expression.
func (dm *AxisExpr) Expr() ast.Expr { return dm.X.Expr() }

// NumAxes returns the number of axis represented by the group.
func (dm *AxisExpr) NumAxes() int { return 1 }

// Equal returns true if two axis are equal.
func (dm *AxisExpr) Equal(tpcmp TypeCmp, other AxisLengths) (bool, error) {
	dmAx, err := tpcmp.EvalExpr(dm.X)
	if err != nil {
		return false, err
	}
	otherAx, err := tpcmp.EvalExpr(other.AsExpr())
	if err != nil {
		return false, err
	}
	eq, err := ElementEqual(dmAx, otherAx)
	if !eq || err != nil {
		return false, err
	}
	return true, nil
}

// Specialise the axis length given a context.
func (dm *AxisExpr) Specialise(spec Specialiser) ([]AxisLengths, bool) {
	x, ok := specialiseExpr(spec, dm.X)
	if !ok {
		return []AxisLengths{&AxisExpr{X: x}}, false
	}
	if x.Type().Kind() != irkind.Slice {
		return []AxisLengths{&AxisExpr{X: x}}, true
	}
	unpacker, canUnpack := x.(ExprUnpacker)
	if !canUnpack {
		return []AxisLengths{&AxisExpr{X: x}}, true
	}
	xs := unpacker.Unpack()
	axlens := make([]AxisLengths, len(xs))
	for i, x := range unpacker.Unpack() {
		axlens[i] = &AxisExpr{X: x}
	}
	return axlens, true
}

// UnifyWith unifies axis lengths with a given target.
func (dm *AxisExpr) UnifyWith(uni Unifier, targets []AxisLengths) ([]AxisLengths, bool) {
	return unifyExpr(uni, targets, dm.X)
}

// Instantiate the rank into another rank.
func (dm *AxisExpr) Instantiate(ev Fetcher, spec Specialiser) ([]AxisLengths, bool) {
	x, ok := CompEvalExpr(ev, spec.Source(), dm.X)
	if x.Type().Kind() != irkind.Slice {
		return []AxisLengths{&AxisExpr{X: x}}, ok
	}
	slice, isSliceLit := x.(*SliceLitExpr)
	if !isSliceLit {
		return []AxisLengths{&AxisExpr{X: x}}, ok
	}
	lens := make([]AxisLengths, len(slice.Elts))
	for i, l := range slice.Elts {
		lens[i] = &AxisExpr{X: l}
	}
	return lens, true
}

// IndexForVarArgs returns a type specific to a given index in varargs.
func (dm *AxisExpr) IndexForVarArgs(i int) AxisLengths {
	return &AxisExpr{X: varArgsIndexExpr(i, dm.X)}
}

// AsExpr returns the axis value as an expression.
func (dm *AxisExpr) AsExpr() Expr { return dm.X }

// Type of the expression.
func (dm *AxisExpr) Type() Type { return dm.X.Type() }

func (dm *AxisExpr) axExprString(from *File) string {
	suffix := ""
	if typ := dm.X.Type(); typ.Kind() == irkind.Slice {
		suffix = "___"
	}
	return dm.X.SourceString(from) + suffix
}

// SourceString returns the GX source code of the axis length.
func (dm *AxisExpr) SourceString(from *File) string {
	return fmt.Sprintf("[%s]", dm.axExprString(from))
}

// AxisInfer is an array axis specified as "_" and inferred by the compiler.
type AxisInfer struct {
	Src *ast.Ident
	X   AxisLengths
}

var _ AxisLengths = (*AxisInfer)(nil)

func (*AxisInfer) node() {}

// Node returns the source expression specifying the axis length.
func (dm *AxisInfer) Node() ast.Node { return dm.Src }

// Type of the expression.
func (dm *AxisInfer) Type() Type { return IntLenType() }

// Expr returns how to compute the expression defining the axis length.
func (dm *AxisInfer) Expr() ast.Expr { return dm.Src }

// AsExpr returns the axis value as an expression.
func (dm *AxisInfer) AsExpr() Expr {
	if dm.X == nil {
		return nil
	}
	return dm.X.AsExpr()
}

// Specialise the axis length given a context.
func (dm *AxisInfer) Specialise(spec Specialiser) ([]AxisLengths, bool) {
	return dm.X.Specialise(spec)
}

// Instantiate the rank into another rank.
func (dm *AxisInfer) Instantiate(Fetcher, Specialiser) ([]AxisLengths, bool) {
	return []AxisLengths{dm}, true
}

// UnifyWith unifies axis lengths with a given target.
func (dm *AxisInfer) UnifyWith(uni Unifier, target []AxisLengths) ([]AxisLengths, bool) {
	return dm.X.UnifyWith(uni, target)
}

// IndexForVarArgs returns a type specific to a given index in varargs.
func (dm *AxisInfer) IndexForVarArgs(i int) AxisLengths {
	if dm.X == nil {
		return nil
	}
	return dm.X.IndexForVarArgs(i)
}

func (dm *AxisInfer) axExprString(from *File) string {
	return dm.X.axExprString(from)
}

// Equal returns true if two axis are equal.
func (dm *AxisInfer) Equal(tpcmp TypeCmp, other AxisLengths) (bool, error) {
	if dm.X == nil {
		return false, nil
	}
	return dm.X.Equal(tpcmp, other)
}

// SourceString returns the GX source code of the axis length.
func (dm *AxisInfer) SourceString(from *File) string {
	if dm.X == nil {
		return "[_]"
	}
	return fmt.Sprintf("[%s]", dm.axExprString(from))
}

// IsAxisSpecType returns true if the type is about an axis,
// that is intlen, intidx, and slices of these types.
func IsAxisSpecType(tp Type) bool {
	switch tp.Kind() {
	case irkind.IntLen, irkind.IntIdx:
		return true
	case irkind.Slice:
		slice, isSlice := Underlying(tp).(*SliceType)
		if !isSlice {
			return false
		}
		return IsAxisSpecType(slice.DType.Val())
	}
	return false
}
