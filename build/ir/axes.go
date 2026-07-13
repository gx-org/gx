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
		IndexForVarArgs(ErrSource, int) ([]AxisLengths, bool)
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

func (dm *AxisExpr) unpackExpr(errsrc ErrSource, x Expr) ([]AxisLengths, bool) {
	unpacker, canUnpack := x.(*UnpackExpr)
	if !canUnpack {
		return []AxisLengths{&AxisExpr{X: x}}, true
	}
	unpacked := unpacker.Unpack()
	switch unpackedT := unpacked.(type) {
	case Expr:
		return []AxisLengths{&AxisExpr{X: unpackedT}}, true
	case *Tuple:
		axlens := make([]AxisLengths, len(unpackedT.Exprs))
		for i, x := range unpackedT.Exprs {
			axlens[i] = &AxisExpr{X: x}
		}
		return axlens, true
	}
	return []AxisLengths{&AxisExpr{X: x}}, errsrc.Err().AppendInternalf(errsrc.Source(), "cannot specialise %s: type %T of the underlying expression not supported", dm.X.SourceString(nil), unpacked)
}

// Specialise the axis length given a context.
func (dm *AxisExpr) Specialise(spec Specialiser) ([]AxisLengths, bool) {
	x, ok := specialiseExpr(spec, dm.X)
	if !ok {
		return []AxisLengths{&AxisExpr{X: x}}, ok
	}
	return dm.unpackExpr(spec, x)
}

// UnifyWith unifies axis lengths with a given target.
func (dm *AxisExpr) UnifyWith(uni Unifier, targets []AxisLengths) ([]AxisLengths, bool) {
	return unifyExpr(uni, targets, dm.X)
}

// Instantiate the rank into another rank.
func (dm *AxisExpr) Instantiate(ev Fetcher, spec Specialiser) ([]AxisLengths, bool) {
	xs, err := CompEvalExpr(ev, dm.X)
	if err != nil {
		return nil, spec.Err().AppendAt(spec.Source(), err)
	}
	axexprs := make([]AxisLengths, len(xs))
	for i, x := range xs {
		axexprs[i] = &AxisExpr{X: x}
	}
	return axexprs, true
}

// IndexForVarArgs returns a type specific to a given index in varargs.
func (dm *AxisExpr) IndexForVarArgs(errsrc ErrSource, i int) ([]AxisLengths, bool) {
	x, ok := varArgsIndexExpr(errsrc, i, dm.X)
	if !ok {
		return []AxisLengths{dm}, ok
	}
	return dm.unpackExpr(errsrc, x)
}

// AsExpr returns the axis value as an expression.
func (dm *AxisExpr) AsExpr() Expr { return dm.X }

// Type of the expression.
func (dm *AxisExpr) Type() Type { return dm.X.Type() }

func (dm *AxisExpr) axExprString(from *File) string {
	return dm.X.SourceString(from)
}

// SourceString returns the GX source code of the axis length.
func (dm *AxisExpr) SourceString(from *File) string {
	lit, isLit := dm.X.(*SliceLitExpr)
	if isLit && len(lit.Elts) == 0 {
		return ""
	}
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
func (dm *AxisInfer) Type() Type { return IntType() }

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
func (dm *AxisInfer) IndexForVarArgs(errsrc ErrSource, i int) ([]AxisLengths, bool) {
	if dm.X == nil {
		return nil, true
	}
	return dm.X.IndexForVarArgs(errsrc, i)
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
