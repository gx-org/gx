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
		axExprString() string

		// Type of the axis.
		Type() Type

		// AsExpr returns the axis value as an expression.
		// Can return nil if the axis has not been resolved.
		AsExpr() Expr

		// Equal returns true if two axis lengths have been resolved and are equal.
		// Returns an error if one of the axis has not been resolved.
		Equal(Fetcher, AxisLengths) (bool, error)

		// AssignableTo returns true if this axis length can be assigned to another.
		AssignableTo(Fetcher, AxisLengths) (bool, error)

		// Specialise the axis length given a context.
		Specialise(Specialiser) ([]AxisLengths, error)

		// UnifyWith unifies axis lengths with a given target.
		UnifyWith(Unifier, []AxisLengths) ([]AxisLengths, bool)

		// String representation of the axis length.
		String() string
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

// AssignableTo returns true if an axis length can be assigned to another.
func (dm *AxisExpr) AssignableTo(fetcher Fetcher, dst AxisLengths) (bool, error) {
	return dm.Equal(fetcher, dst)
}

// Equal returns true if other has the axis length.
func (dm *AxisExpr) Equal(fetcher Fetcher, other AxisLengths) (bool, error) {
	return areEqual(fetcher, dm.X, other.AsExpr())
}

// Specialise the axis length given a context.
func (dm *AxisExpr) Specialise(spec Specialiser) ([]AxisLengths, error) {
	el, err := spec.EvalExpr(dm.X)
	if err != nil {
		return []AxisLengths{dm}, err
	}
	exprIR, err := toExpr(el)
	if err != nil {
		return []AxisLengths{dm}, err
	}
	sliceLit, isSliceLit := exprIR.(*SliceLitExpr)
	if !isSliceLit {
		return []AxisLengths{&AxisExpr{X: exprIR}}, nil
	}
	axes := make([]AxisLengths, len(sliceLit.Elts))
	for i, expr := range sliceLit.Elts {
		axes[i] = &AxisExpr{X: expr}
	}
	return axes, nil
}

// UnifyWith unifies axis lengths with a given target.
func (dm *AxisExpr) UnifyWith(uni Unifier, targets []AxisLengths) ([]AxisLengths, bool) {
	return []AxisLengths{dm}, true
}

// AsExpr returns the axis value as an expression.
func (dm *AxisExpr) AsExpr() Expr { return dm.X }

// Type of the expression.
func (dm *AxisExpr) Type() Type { return dm.X.Type() }

func (dm *AxisExpr) axExprString() string {
	suffix := ""
	if typ := dm.X.Type(); typ.Kind() == irkind.Slice {
		suffix = "___"
	}
	return dm.X.String() + suffix
}

// String representation of the axis length.
func (dm *AxisExpr) String() string {
	return fmt.Sprintf("[%s]", dm.axExprString())
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

// Equal returns true if other has the axis length.
func (dm *AxisInfer) Equal(fetcher Fetcher, other AxisLengths) (bool, error) {
	return areEqual(fetcher, dm.AsExpr(), other.AsExpr())
}

// AssignableTo returns true if a dimension can be assigned to another.
func (dm *AxisInfer) AssignableTo(fetcher Fetcher, dst AxisLengths) (bool, error) {
	return dm.Equal(fetcher, dst)
}

// AsExpr returns the axis value as an expression.
func (dm *AxisInfer) AsExpr() Expr {
	if dm.X == nil {
		return nil
	}
	return dm.X.AsExpr()
}

// Specialise the axis length given a context.
func (dm *AxisInfer) Specialise(spec Specialiser) ([]AxisLengths, error) {
	return dm.X.Specialise(spec)
}

// UnifyWith unifies axis lengths with a given target.
func (dm *AxisInfer) UnifyWith(uni Unifier, target []AxisLengths) ([]AxisLengths, bool) {
	return dm.X.UnifyWith(uni, target)
}

func (dm *AxisInfer) axExprString() string {
	return dm.X.axExprString()
}

// String representation of the dimension.
func (dm *AxisInfer) String() string {
	if dm.X == nil {
		return "[_]"
	}
	return fmt.Sprintf("[%s]", dm.axExprString())
}

// AxisStmt is an array axis specified using a statement.
type AxisStmt struct {
	Src *ast.Ident
	Typ Type
}

var (
	_ AxisLengths = (*AxisStmt)(nil)
	_ Storage     = (*AxisStmt)(nil)
)

func (*AxisStmt) node()    {}
func (*AxisStmt) storage() {}

// Node returns the source expression specifying the axis length.
func (dm *AxisStmt) Node() ast.Node { return dm.Src }

// NameDef returns the identifier identifying the storage.
func (dm *AxisStmt) NameDef() *ast.Ident { return dm.Src }

// NumAxes returns the number of axis represented by the group.
func (dm *AxisStmt) NumAxes() int { return 1 }

// AssignableTo returns true if an axis length can be assigned to another.
func (dm *AxisStmt) AssignableTo(fetcher Fetcher, dst AxisLengths) (bool, error) {
	return dm.Equal(fetcher, dst)
}

// Equal returns true if other has the axis length.
func (dm *AxisStmt) Equal(fetcher Fetcher, other AxisLengths) (bool, error) {
	return areEqual(fetcher, dm.AsExpr(), other.AsExpr())
}

// Same returns true if the other storage is this storage.
func (dm *AxisStmt) Same(o Storage) bool {
	return Storage(dm) == o
}

// Specialise the axis length given a context.
func (dm *AxisStmt) Specialise(spec Specialiser) ([]AxisLengths, error) {
	el, err := spec.EvalExpr(dm.AsExpr())
	if err != nil {
		return []AxisLengths{dm}, err
	}
	exprIR, err := toExpr(el)
	if err != nil {
		return []AxisLengths{dm}, err
	}
	switch exprT := exprIR.(type) {
	case *SliceLitExpr:
		axes := make([]AxisLengths, len(exprT.Elts))
		for i, expr := range exprT.Elts {
			axes[i] = &AxisExpr{X: expr}
		}
		return axes, nil
	case WithStore:
		ax, isAxisStmt := exprT.Store().(*AxisStmt)
		if isAxisStmt {
			return []AxisLengths{ax}, nil
		}
	}
	return []AxisLengths{&AxisExpr{X: exprIR}}, nil
}

// UnifyWith unifies axis lengths with a given target.
func (dm *AxisStmt) UnifyWith(uni Unifier, targets []AxisLengths) ([]AxisLengths, bool) {
	return uni.DefineAxis(dm, targets)
}

// AsExpr returns the value assigned to the axis.
func (dm *AxisStmt) AsExpr() Expr {
	return &Ident{Src: dm.Src, Stor: dm}
}

// Type of the expression.
func (dm *AxisStmt) Type() Type {
	return dm.Typ
}

func (dm *AxisStmt) axExprString() string {
	prefix := DefineAxisLength
	if dm.Typ.Kind() == irkind.Slice {
		prefix = DefineAxisGroup
	}
	return prefix + dm.Src.Name
}

// String representation of the axis length.
func (dm *AxisStmt) String() string {
	return fmt.Sprintf("[%s]", dm.axExprString())
}
