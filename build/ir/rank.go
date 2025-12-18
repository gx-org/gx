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
		Equal(Fetcher, ArrayRank) (bool, error)

		// AssignableTo returns true if this rank can be assigned to the destination rank.
		AssignableTo(Fetcher, ArrayRank) (bool, error)

		// ConvertibleTo returns true if this rank can be converted to the destination rank.
		ConvertibleTo(Fetcher, ArrayRank) (bool, error)

		// Specialise a type to a given target.
		Specialise(Specialiser) (ArrayRank, error)

		// UnifyWith unifies the rank with a given target.
		UnifyWith(Unifier, ArrayRank) bool

		// SubRank returns the rank with the top-axis removed.
		SubRank() (ArrayRank, bool)

		// String representation of the rank.
		String() string
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

// Source returns the source node defining the rank.
func (r *Rank) Source() ast.Node { return r.Src }

// Axes returns all axis in the rank.
func (r *Rank) Axes() []AxisLengths { return r.Ax }

// Equal returns true if other has the same number of axes and each axis has the same length.
func (r *Rank) Equal(fetcher Fetcher, other ArrayRank) (bool, error) {
	switch otherT := other.(type) {
	case *RankInfer:
		if otherT.Rnk == nil {
			return false, errors.Errorf("rank not inferred")
		}
		return r.equalRank(fetcher, otherT.Rnk)
	case *Rank:
		return r.equalRank(fetcher, otherT)
	default:
		return false, errors.Errorf("rank type %T not supported", otherT)
	}
}

func (r *Rank) equalRank(fetcher Fetcher, other ArrayRank) (bool, error) {
	// Check the number of axes.
	otherAx := other.Axes()
	if len(r.Ax) != len(otherAx) {
		return false, nil
	}
	// Check each axis.
	for i, dimX := range r.Ax {
		dimEq, err := dimX.Equal(fetcher, otherAx[i])
		if err != nil {
			return false, err
		}
		if !dimEq {
			return false, nil
		}
	}
	return true, nil
}

func (r *Rank) assignableTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	dstAx := dst.Axes()
	if len(r.Ax) != len(dstAx) {
		return false, nil
	}
	for i, dimThis := range r.Ax {
		ok, err := dimThis.AssignableTo(fetcher, dstAx[i])
		if !ok || err != nil {
			return ok, err
		}
	}
	return true, nil
}

// AssignableTo returns true if this rank can be assigned to the destination rank.
func (r *Rank) AssignableTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	switch dstT := dst.(type) {
	case *RankInfer:
		if dstT.Rnk == nil {
			return true, nil
		}
		return r.assignableTo(fetcher, dstT.Rnk)
	case *Rank:
		return r.assignableTo(fetcher, dstT)
	default:
		return false, errors.Errorf("rank type %T not supported", dstT)
	}
}

// ConvertibleTo returns true if this rank can be converted to the destination rank.
func (r *Rank) ConvertibleTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
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
	return areEqual(fetcher, thisSize, otherSize)
}

// Specialise a type to a given target.
func (r *Rank) Specialise(spec Specialiser) (ArrayRank, error) {
	var axes []AxisLengths
	for _, ax := range r.Ax {
		subs, err := ax.Specialise(spec)
		if err != nil {
			return r, err
		}
		axes = append(axes, subs...)
	}
	return &Rank{Src: r.Src, Ax: axes}, nil
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

var oneSize = &NumberCastExpr{
	X:   &NumberInt{Val: big.NewInt(1)},
	Typ: IntLenType(),
}

func exprSource(n SourceNode) ast.Expr {
	if n == nil {
		return nil
	}
	src := n.Source()
	if src == nil {
		return nil
	}
	expr, _ := src.(ast.Expr)
	return expr
}

// RankSize is the total number of elements across all axes.
func RankSize(r ArrayRank) Expr {
	var expr AssignableExpr = oneSize
	for _, axis := range r.Axes() {
		expr = &BinaryExpr{
			Src: &ast.BinaryExpr{
				Op: token.MUL,
				X:  exprSource(expr),
				Y:  exprSource(axis),
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

// String returns a string representation of the rank.
func (r *Rank) String() string {
	if r == nil {
		return ""
	}
	bld := strings.Builder{}
	for _, dim := range r.Ax {
		bld.WriteString(dim.String())
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

// Source returns the node defining the rank.
func (r *RankInfer) Source() ast.Node { return r.Src }

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
func (r *RankInfer) Equal(fetcher Fetcher, other ArrayRank) (bool, error) {
	if r.Rnk != nil {
		return r.Rnk.Equal(fetcher, other)
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
func (r *RankInfer) AssignableTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	if r.Rnk != nil {
		return r.Rnk.AssignableTo(fetcher, dst)
	}
	return true, nil
}

// ConvertibleTo returns true if this rank can be converted to the destination rank.
func (r *RankInfer) ConvertibleTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	if r.Rnk != nil {
		return r.Rnk.ConvertibleTo(fetcher, dst)
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
func (r *RankInfer) Specialise(spec Specialiser) (ArrayRank, error) {
	if r.Rnk == nil {
		return r, nil
	}
	return r.Rnk.Specialise(spec)
}

// UnifyWith unifies the rank with a given target.
func (r *RankInfer) UnifyWith(uni Unifier, target ArrayRank) bool {
	if r.Rnk == nil {
		return true
	}
	return r.Rnk.UnifyWith(uni, target)
}

// String returns a string representation of the rank.
func (r *RankInfer) String() string {
	if r.Rnk == nil {
		return "[...]"
	}
	return r.Rnk.String()
}
