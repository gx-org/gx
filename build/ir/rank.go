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
	"go/ast"
	"go/token"
	"math/big"
	"strings"

	"github.com/pkg/errors"
)

type (
	// ArrayRank of an array.
	ArrayRank interface {
		Expr
		nodeRank()

		// IsAtomic returns true if the rank has no axes.
		IsAtomic() bool

		// Axes returns all axis in the rank.
		Axes() []AxisLengths

		// Equal returns true if two ranks are equal.
		Equal(TypeCmp, ArrayRank) (bool, error)

		// AssignableTo returns true if this rank can be assigned to the destination rank.
		AssignableTo(TypeCmp, ArrayRank) (bool, error)

		// ConvertibleTo returns true if this rank can be converted to the destination rank.
		ConvertibleTo(TypeCmp, ArrayRank) (bool, error)

		// Specialise a type to a given target.
		Specialise(Specialiser) ArrayRank

		// UnifyWith unifies the rank with a given target.
		UnifyWith(Unifier, ArrayRank) bool

		// Instantiate the rank into another rank.
		Instantiate(Fetcher, Specialiser) (ArrayRank, bool)

		// IndexForVarArgs returns a rank specific to a given index in varargs.
		IndexForVarArgs(i int) ArrayRank

		// SubRank returns the rank with the top-axis removed.
		SubRank() (ArrayRank, bool)
	}

	// Rank with a known number of axes.
	// The length of some axes may still be unknown.
	Rank struct {
		Src *ast.ArrayType
		Ax  []AxisLengths
	}
)

var _ ArrayRank = (*Rank)(nil)

func (*Rank) node()     {}
func (*Rank) nodeRank() {}

// NewRank returns a new rank from a slice of axis lengths.
func NewRank(axlens []int) *Rank {
	axes := make([]AxisLengths, len(axlens))
	for i, al := range axlens {
		axes[i] = &AxisExpr{
			X: &NumberCastExpr{
				X:   &NumberInt{Val: big.NewInt(int64(al))},
				Typ: IntLenType(),
			},
		}
	}
	return &Rank{Ax: axes}
}

// Type returns the rank type.
func (r *Rank) Type() Type { return RankType() }

// Node returns the source node defining the rank.
func (r *Rank) Node() ast.Node { return r.Src }

// Expr returns the AST expression.
func (r *Rank) Expr() ast.Expr { return r.Src.Len }

// Axes returns all axis in the rank.
func (r *Rank) Axes() []AxisLengths { return r.Ax }

// Equal returns true if other has the same number of axes and each axis has the same length.
func (r *Rank) Equal(tpcmp TypeCmp, other ArrayRank) (bool, error) {
	switch otherT := other.(type) {
	case *RankInfer:
		if otherT.Rnk == nil {
			return false, errors.Errorf("rank not inferred")
		}
		return r.equalRank(tpcmp, otherT.Rnk)
	case *Rank:
		return r.equalRank(tpcmp, otherT)
	default:
		return false, errors.Errorf("rank type %T not supported", otherT)
	}
}

func evalAxes(tpcmp TypeCmp, r ArrayRank) ([]Element, error) {
	var els []Element
	for _, ax := range r.Axes() {
		el, err := tpcmp.EvalExpr(ax.AsExpr())
		if err != nil {
			return nil, err
		}
		axElts := []Element{el}
		if tuple, isTuple := el.(TupleElement); isTuple {
			axElts = tuple.TupleElements()
		}
		els = append(els, axElts...)
	}
	return els, nil
}

func (r *Rank) equalRank(tpcmp TypeCmp, other ArrayRank) (bool, error) {
	rAxes, err := evalAxes(tpcmp, r)
	if err != nil {
		return false, err
	}
	otherAxes, err := evalAxes(tpcmp, other)
	if err != nil {
		return false, err
	}
	if len(rAxes) != len(otherAxes) {
		return false, err
	}
	for i, ri := range rAxes {
		eq, err := ElementEqual(ri, otherAxes[i])
		if !eq || err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *Rank) assignableTo(tpcmp TypeCmp, dst ArrayRank) (bool, error) {
	return r.equalRank(tpcmp, dst)
}

// AssignableTo returns true if this rank can be assigned to the destination rank.
func (r *Rank) AssignableTo(tpcmp TypeCmp, dst ArrayRank) (bool, error) {
	switch dstT := dst.(type) {
	case *RankInfer:
		if dstT.Rnk == nil {
			return true, nil
		}
		return r.assignableTo(tpcmp, dstT.Rnk)
	case *Rank:
		return r.assignableTo(tpcmp, dstT)
	default:
		return false, errors.Errorf("rank type %T not supported", dstT)
	}
}

// ConvertibleTo returns true if this rank can be converted to the destination rank.
func (r *Rank) ConvertibleTo(tpcmp TypeCmp, dst ArrayRank) (bool, error) {
	var dstR ArrayRank
	switch dstT := dst.(type) {
	case *RankInfer:
		if dstT.Rnk == nil {
			return true, nil
		}
		dstR = dstT.Rnk
	case *Rank:
		dstR = dstT
	default:
		return false, errors.Errorf("rank type %T not supported", dstT)
	}
	thisSize := RankSize(r)
	otherSize := RankSize(dstR)
	return areEqual(tpcmp, thisSize, otherSize)
}

// Specialise a type to a given target.
func (r *Rank) Specialise(spec Specialiser) ArrayRank {
	var axes []AxisLengths
	for _, ax := range r.Ax {
		subs := ax.Specialise(spec)
		axes = append(axes, subs...)
	}
	return &Rank{Src: r.Src, Ax: axes}
}

// UnifyWith unifies the rank with a given target.
func (r *Rank) UnifyWith(uni Unifier, arg ArrayRank) bool {
	targets := arg.Axes()
	for _, ax := range r.Ax {
		var ok bool
		targets, ok = ax.UnifyWith(uni, targets)
		if !ok {
			return false
		}
	}
	return true
}

// Instantiate the rank into another rank.
func (r *Rank) Instantiate(ev Fetcher, spec Specialiser) (ArrayRank, bool) {
	var axes []AxisLengths
	ok := true
	for _, ax := range r.Ax {
		subs, axOk := ax.Instantiate(ev, spec)
		axes = append(axes, subs...)
		ok = ok && axOk
	}
	return &Rank{Src: r.Src, Ax: axes}, ok
}

// IndexForVarArgs returns a rank specific to a given index in varargs.
func (r *Rank) IndexForVarArgs(vri int) ArrayRank {
	axes := make([]AxisLengths, len(r.Ax))
	for i, ax := range r.Ax {
		axes[i] = ax.IndexForVarArgs(vri)
	}
	return &Rank{Src: r.Src, Ax: axes}
}

var oneSize = &NumberCastExpr{
	X:   &NumberInt{Val: big.NewInt(1)},
	Typ: IntLenType(),
}

// RankSize is the total number of elements across all axes.
func RankSize(r ArrayRank) Expr {
	var expr Expr = oneSize
	for _, axis := range r.Axes() {
		expr = &BinaryExpr{
			Src: &ast.BinaryExpr{
				Op: token.MUL,
				X:  expr.Expr(),
				Y:  axis.AsExpr().Expr(),
			},
			X:   expr,
			Y:   axis.AsExpr(),
			Typ: IntLenType(),
		}
	}
	return expr
}

// IsAtomic returns true if the rank is equals to zero
// (that is it has no axes).
func (r *Rank) IsAtomic() bool {
	return len(r.Ax) == 0
}

// AxisLengths of the rank.
func (r *Rank) AxisLengths() []AxisLengths {
	return r.Ax
}

// SubRank returns the rank with the top-axis removed
// or nil if the rank is already 0.
func (r *Rank) SubRank() (ArrayRank, bool) {
	if len(r.Ax) == 0 {
		return nil, false
	}
	return &Rank{Ax: append([]AxisLengths{}, r.Ax[1:]...)}, true
}

// SourceString returns the GX source code of the rank.
func (r *Rank) SourceString(from *File) string {
	if r == nil {
		return ""
	}
	bld := strings.Builder{}
	for _, dim := range r.Ax {
		bld.WriteString(dim.SourceString(from))
	}
	return bld.String()
}

// RankInfer is a rank determined at compile time
// (specified using ___ or ...).
type RankInfer struct {
	Src *ast.ArrayType
	Rnk ArrayRank
}

var _ ArrayRank = (*RankInfer)(nil)

func (*RankInfer) node()     {}
func (*RankInfer) nodeRank() {}

// Node returns the node defining the rank.
func (r *RankInfer) Node() ast.Node { return r.Expr() }

// Expr returns the AST expression.
func (r *RankInfer) Expr() ast.Expr { return r.Src.Len }

// Type returns the rank type.
func (r *RankInfer) Type() Type { return RankType() }

// Axes returns all axis in the rank.
func (r *RankInfer) Axes() []AxisLengths {
	if r.Rnk == nil {
		return nil
	}
	return r.Rnk.Axes()
}

// Equal returns true if other has the same rank and dimensions.
func (r *RankInfer) Equal(tpcmp TypeCmp, other ArrayRank) (bool, error) {
	if r.Rnk != nil {
		return r.Rnk.Equal(tpcmp, other)
	}
	switch otherT := other.(type) {
	case *RankInfer:
		return otherT.Rnk == nil, nil
	case *Rank:
		return false, nil
	default:
		return false, errors.Errorf("rank type %T not supported", otherT)
	}
}

// AssignableTo returns true if this rank can be assigned to the destination rank.
func (r *RankInfer) AssignableTo(tpcmp TypeCmp, dst ArrayRank) (bool, error) {
	if r.Rnk != nil {
		return r.Rnk.AssignableTo(tpcmp, dst)
	}
	return true, nil
}

// ConvertibleTo returns true if this rank can be converted to the destination rank.
func (r *RankInfer) ConvertibleTo(tpcmp TypeCmp, dst ArrayRank) (bool, error) {
	if r.Rnk != nil {
		return r.Rnk.ConvertibleTo(tpcmp, dst)
	}
	return true, nil
}

// IsAtomic returns true if the rank has no axes.
func (r *RankInfer) IsAtomic() bool {
	return false
}

// SubRank returns the rank with the top-axis removed.
func (r *RankInfer) SubRank() (ArrayRank, bool) {
	if r.Rnk == nil {
		return &RankInfer{}, true
	}
	return r.Rnk.SubRank()
}

// Specialise a type to a given target.
func (r *RankInfer) Specialise(spec Specialiser) ArrayRank {
	if r.Rnk == nil {
		return r
	}
	return r.Rnk.Specialise(spec)
}

// Instantiate the rank into another rank.
func (r *RankInfer) Instantiate(ev Fetcher, spec Specialiser) (ArrayRank, bool) {
	if r.Rnk == nil {
		return r, true
	}
	return r.Rnk.Instantiate(ev, spec)
}

// UnifyWith unifies the rank with a given target.
func (r *RankInfer) UnifyWith(uni Unifier, target ArrayRank) bool {
	if r.Rnk == nil {
		return true
	}
	return r.Rnk.UnifyWith(uni, target)
}

// IndexForVarArgs returns a rank specific to a given index in varargs.
func (r *RankInfer) IndexForVarArgs(i int) ArrayRank {
	if r.Rnk == nil {
		return r
	}
	return r.Rnk.IndexForVarArgs(i)
}

// SourceString returns the GX source code of the rank.
func (r *RankInfer) SourceString(from *File) string {
	if r.Rnk == nil {
		return "[...]"
	}
	return r.Rnk.SourceString(from)
}
