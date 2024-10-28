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

package builder

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir"
)

type (
	rankNode interface {
		build() ir.ArrayRank
		subRank() rankNode
		reconcileWith(scoper, nodePos, rankNode) (rankNode, bool)
		resolve(scope scoper) bool
		numAxes() int
		String() string
	}

	rank struct {
		ext  ir.Rank
		dims []arrayAxes
	}

	genericRank struct {
		ext ir.GenericRank
		rnk *rank
	}

	anyRank struct {
		ext ir.AnyRank
	}
)

var (
	_ rankNode = (*rank)(nil)
	_ rankNode = (*genericRank)(nil)
	_ rankNode = (*anyRank)(nil)
)

const cannotInfer = "cannot infer number of array axes"

func processDTypeRank(owner owner, typ *ast.ArrayType) (rankNode, typeNode, bool) {
	dims := []arrayAxes{}
	var dtype typeNode
	var rnk rankNode

	ok := true
	var elt ast.Node = typ
	for dtype == nil {
		switch eltT := elt.(type) {
		case *ast.Ident:
			var dtypeOk bool
			dtype, dtypeOk = processTypeExpr(owner, eltT)
			ok = ok && dtypeOk && eltT != nil
			if rnk != nil {
				return rnk, dtype, ok
			}
		case *ast.ArrayType:
			if rnk != nil {
				owner.err().Appendf(elt, "array type can only have one unknown axis")
				return nil, invalid, false
			}
			dim, dimOk := processArrayDimensionExpr(owner, eltT)
			if !dimOk {
				return nil, invalid, false
			}
			if dim == nil {
				rnk = &genericRank{ext: ir.GenericRank{Src: eltT}}
			} else {
				dims = append(dims, dim)
			}
			elt = eltT.Elt
		default:
			owner.err().Appendf(elt, "element type %T in array declaration not supported", elt)
			return nil, invalid, false
		}
	}
	return &rank{
		ext:  ir.Rank{Src: typ},
		dims: dims,
	}, dtype, ok
}

func importRank(scope scoper, ext ir.ArrayRank) (rankNode, bool) {
	var r *rank
	switch extT := ext.(type) {
	case *ir.AnyRank:
		return &anyRank{ext: *extT}, true
	case *ir.GenericRank:
		return &genericRank{ext: *extT}, true
	case *ir.Rank:
		r = &rank{ext: *extT}
	}
	ok := true
	r.dims = make([]arrayAxes, len(r.ext.Axes))
	for i, dim := range r.ext.Axes {
		var dimOk bool
		r.dims[i], dimOk = importAxis(scope, dim)
		ok = ok && dimOk
	}
	return r, ok
}

func (r *rank) buildRank() *ir.Rank {
	r.ext.Axes = make([]ir.AxisLength, len(r.dims))
	for i, dim := range r.dims {
		r.ext.Axes[i] = dim.expr()
	}
	return &r.ext
}

func (r *rank) build() ir.ArrayRank {
	return r.buildRank()
}

func (r *rank) subRankR() *rank {
	return &rank{
		ext:  ir.Rank{Src: r.ext.Src},
		dims: r.dims[1:],
	}
}

func (r *rank) subRank() rankNode {
	return r.subRankR()
}

func (r *rank) numAxes() int {
	return len(r.dims)
}

func lengthWord(r rankNode) string {
	if r.numAxes() > 1 {
		return "lengths"
	}
	return "length"
}

func reconcileRankWith(scope scoper, pos nodePos, src, other *rank) (*rank, bool) {
	if len(src.dims) != len(other.dims) {
		scope.err().Appendf(pos.source(), "cannot reconcile a %d-axis array with a %d-axis array", len(src.dims), len(other.dims))
		return src, false
	}
	dst := &rank{
		ext:  src.ext,
		dims: make([]arrayAxes, len(src.dims)),
	}
	ok := true
	for i, dim := range src.dims {
		var dimOk bool
		dst.dims[i], dimOk = dim.reconcileWith(scope, other.dims[i])
		ok = dimOk && ok
	}
	if !ok {
		scope.err().Appendf(pos.source(), "cannot reconcile axis %s %s with axis %s %s", lengthWord(src), src.build().String(), lengthWord(other), other.build().String())
	}
	return dst, ok

}

func (r *rank) reconcileWith(scope scoper, pos nodePos, other rankNode) (rankNode, bool) {
	switch otherT := other.(type) {
	case *rank:
		return reconcileRankWith(scope, pos, r, otherT)
	case *genericRank:
		if otherT.rnk == nil {
			return r, true
		}
		return reconcileRankWith(scope, pos, r, otherT.rnk)
	default:
		scope.err().Appendf(pos.source(), "cannot reconcile rank with %T", other)
		return r, false
	}
}

func (r *rank) resolve(scope scoper) bool {
	ok := true
	for _, dim := range r.dims {
		ok = dim.resolve(scope) && ok
	}
	return ok
}

func (r *rank) String() string {
	return r.build().String()
}

func (r *anyRank) build() ir.ArrayRank {
	return &r.ext
}

func (r *anyRank) numAxes() int {
	return -1
}

func (r *anyRank) subRank() rankNode {
	return r
}

func (r *anyRank) reconcileWith(scope scoper, pos nodePos, other rankNode) (rankNode, bool) {
	scope.err().Appendf(pos.source(), "cannot reconcile an array with an unknown number of axes. Use [...] instead of [any].")
	return r, false
}

func (r *anyRank) resolve(scope scoper) bool {
	return true
}

func (r *anyRank) String() string {
	return r.build().String()
}

func (r *genericRank) build() ir.ArrayRank {
	if r.rnk != nil {
		r.ext.Rnk = r.rnk.buildRank()
	}
	return &r.ext
}

func (r *genericRank) numAxes() int {
	return -1
}

func (r *genericRank) subRank() rankNode {
	sub := &genericRank{
		ext: ir.GenericRank{Src: r.ext.Src},
	}
	if r.rnk == nil {
		return sub
	}
	sub.rnk = r.rnk.subRankR()
	return sub
}

func (r *genericRank) newResolvedRank(src *rank) *rank {
	resolved := &rank{
		ext:  ir.Rank{Src: src.ext.Src},
		dims: make([]arrayAxes, len(src.dims)),
	}
	for i := range resolved.dims {
		resolved.dims[i] = &genericDimension{
			ext: &ir.AxisEllipsis{Src: r.ext.Src.Elt.(*ast.Ident)},
		}
	}
	return resolved
}

func (r *genericRank) reconcileWith(scope scoper, pos nodePos, other rankNode) (rankNode, bool) {
	switch otherT := other.(type) {
	case *rank:
		toReconcile := r.newResolvedRank(otherT)
		rank, ok := reconcileRankWith(scope, pos, toReconcile, otherT)
		rank.ext.Src = r.ext.Src
		return &genericRank{
			ext: ir.GenericRank{
				Src: r.ext.Src,
			},
			rnk: rank,
		}, ok
	case *genericRank:
		if otherT.rnk == nil {
			return r, true
		}
		return r.reconcileWith(scope, pos, otherT.rnk)
	case *anyRank:
		return otherT.reconcileWith(scope, pos, r)
	default:
		return r, scope.err().AppendInternalf(pos.source(), "cannot reconcile rank with %T", other)
	}
}

func (r *genericRank) resolve(scope scoper) bool {
	return true
}

func (r *genericRank) String() string {
	return r.build().String()
}
