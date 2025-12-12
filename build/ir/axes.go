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
	"strings"

	"github.com/pkg/errors"
)

const (
	// DefineAxisGroup is the prefix used in parameters to define an axis group.
	DefineAxisGroup = "___"
	// DefineAxisLength is the prefix used in parameters to define an axis length.
	DefineAxisLength = "_"
)

// GenAxLenName defines a name for the length of a generic axis or a generic group of axes.
type GenAxLenName struct {
	Src   *ast.Ident
	Value Element
	Typ   Type
}

var _ Storage = (*GenAxLenName)(nil)

func (*GenAxLenName) node()    {}
func (*GenAxLenName) storage() {}

// Source returns the node in the AST tree.
func (s *GenAxLenName) Source() ast.Node {
	return s.Src
}

// Type of the destination of the assignment.
func (s *GenAxLenName) Type() Type { return s.Typ }

// NameDef returns the identifier identifying the storage.
func (s *GenAxLenName) NameDef() *ast.Ident { return s.Src }

// Name returns the name of the axis.
func (s *GenAxLenName) Name() string {
	return strings.TrimPrefix(s.Src.Name, DefineAxisGroup)
}

// Same returns true if the other storage is this storage.
func (s *GenAxLenName) Same(o Storage) bool {
	return Storage(s) == o
}

func (s *GenAxLenName) axExprString() string {
	return s.Src.Name
}

// String representation of the dimension.
func (s *GenAxLenName) String() string {
	return fmt.Sprintf("[%s]", s.axExprString())
}

type (
	// AxisLengths specification of an array.
	AxisLengths interface {
		axExprString() string
		Expr

		// AxisValue returns the value associated with the axis.
		// Can return nil if the axis has not been resolved.
		AxisValue() AssignableExpr

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
		// Source of the axis expression.
		// May be different from the source of the expression, for example
		// the expression is formed from a function call.
		Src ast.Expr
		// X computes the size of the axis.
		X AssignableExpr
	}
)

var _ AxisLengths = (*AxisExpr)(nil)

func (*AxisExpr) node() {}

// Source returns the source expression specifying the dimension.
func (dm *AxisExpr) Source() ast.Node { return dm.X.Source() }

// NumAxes returns the number of axis represented by the group.
func (dm *AxisExpr) NumAxes() int { return 1 }

// AssignableTo returns true if a dimension can be assigned to another.
func (dm *AxisExpr) AssignableTo(fetcher Fetcher, dst AxisLengths) (bool, error) {
	switch dstT := dst.(type) {
	case *AxisExpr:
		return dm.Equal(fetcher, dst)
	case *AxisInfer:
		if dstT.X == nil {
			return true, nil
		}
		return dm.Equal(fetcher, dstT.X)
	default:
		return false, errors.Errorf("assigning an axis to an axis type %T not supported", dstT)
	}
}

// Equal returns true if other has the axis length.
func (dm *AxisExpr) Equal(fetcher Fetcher, other AxisLengths) (bool, error) {
	switch otherT := other.(type) {
	case *AxisExpr:
		return areEqual(fetcher, dm.X, otherT.X)
	case *AxisInfer:
		if otherT.X == nil {
			return false, errors.Errorf("cannot compare with an unresolved axis length")
		}
		return dm.Equal(fetcher, otherT.X)
	default:
		return false, errors.Errorf("cannot compare with axis type %T: not supported", otherT)
	}
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
		return []AxisLengths{&AxisExpr{Src: dm.Src, X: exprIR}}, nil
	}
	axes := make([]AxisLengths, len(sliceLit.Elts))
	for i, expr := range sliceLit.Elts {
		axes[i] = &AxisExpr{Src: dm.Src, X: expr}
	}
	return axes, nil
}

// UnifyWith unifies axis lengths with a given target.
func (dm *AxisExpr) UnifyWith(uni Unifier, targets []AxisLengths) ([]AxisLengths, bool) {
	return uni.DefineAxis(dm, targets)
}

// AxisValue returns the value assigned to the axis.
func (dm *AxisExpr) AxisValue() AssignableExpr { return dm.X }

// Value of the axis.
func (dm *AxisExpr) Value(Expr) AssignableExpr { return dm.AxisValue() }

// Type of the expression.
func (dm *AxisExpr) Type() Type { return dm.X.Type() }

func (dm *AxisExpr) axExprString() string {
	return dm.X.String()
}

// String representation of the dimension.
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

// Source returns the source expression specifying the dimension.
func (dm *AxisInfer) Source() ast.Node { return dm.Src }

// Type of the expression.
func (dm *AxisInfer) Type() Type { return IntLenType() }

// Expr returns how to compute the dimension as an expression.
func (dm *AxisInfer) Expr() ast.Expr { return dm.Src }

// Equal returns true if other has the axis length.
func (dm *AxisInfer) Equal(fetcher Fetcher, other AxisLengths) (bool, error) {
	switch otherT := other.(type) {
	case *AxisExpr:
		if dm.X == nil {
			return false, errors.Errorf("unresolved axis length")
		}
		return dm.X.Equal(fetcher, otherT)
	case *AxisInfer:
		if dm.X == nil && otherT.X == nil {
			return true, nil
		}
		if dm.X != nil && otherT.X != nil {
			return dm.X.Equal(fetcher, otherT.X)
		}
		return false, nil
	default:
		return false, errors.Errorf("cannot compare with axis type %T: not supported", otherT)
	}
}

// AssignableTo returns true if a dimension can be assigned to another.
func (dm *AxisInfer) AssignableTo(fetcher Fetcher, dst AxisLengths) (bool, error) {
	if dm.X != nil {
		return dm.X.AssignableTo(fetcher, dst)
	}
	return true, nil
}

// AxisValue returns the value assigned to the axis.
func (dm *AxisInfer) AxisValue() AssignableExpr {
	if dm.X == nil {
		return nil
	}
	return dm.X.AxisValue()
}

// Specialise the axis length given a context.
func (dm *AxisInfer) Specialise(spec Specialiser) ([]AxisLengths, error) {
	return dm.X.Specialise(spec)
}

// UnifyWith unifies axis lengths with a given target.
func (dm *AxisInfer) UnifyWith(uni Unifier, target []AxisLengths) ([]AxisLengths, bool) {
	return dm.X.UnifyWith(uni, target)
}

// Value of the axis.
func (dm *AxisInfer) Value(Expr) AssignableExpr {
	return dm.AxisValue()
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
