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
	"strings"

	"github.com/gx-org/gx/build/ir"
)

type (
	rankNode interface {
		build(localScope) (ir.ArrayRank, bool)
		String() string
	}

	rank struct {
		src  *ast.ArrayType
		dims []axisLengthNode
	}
)

var _ rankNode = (*rank)(nil)

func processDTypeRank(pscope procScope, src *ast.ArrayType) (rankNode, typeExprNode, bool) {
	var ranks []rankNode
	var axes []axisLengthNode
	var dtype typeExprNode

	ok := true
	var elt ast.Node = src
	axscope := pscope.axisLengthScope()
	for dtype == nil {
		switch eltT := elt.(type) {
		case *ast.Ident:
			var dtypeOk bool
			dtype, dtypeOk = processTypeExpr(pscope, eltT)
			ok = ok && dtypeOk && eltT != nil
		case *ast.ArrayType:
			axis, axisOk := processAxisLengthExpr(axscope, eltT)
			if !axisOk {
				return nil, nil, false
			}
			if axis == nil {
				// Directive to infer the rank from the array.
				ranks = append(ranks, &genericRank{src: eltT})
			} else {
				axes = append(axes, axis)
			}
			elt = eltT.Elt
		default:
			return nil, nil, pscope.Err().Appendf(elt, "element type %T in array declaration not supported", elt)
		}
	}
	if len(ranks) == 0 {
		return &rank{src: src, dims: axes}, dtype, ok
	}
	if len(axes) > 0 || len(ranks) > 1 {
		ok = pscope.Err().Appendf(src, "array with an inferred rank only support a single axis")
	}
	return ranks[0], dtype, ok
}

func (r *rank) build(rscope localScope) (ir.ArrayRank, bool) {
	ext := &ir.Rank{
		Src: r.src,
		Ax:  make([]ir.AxisLengths, len(r.dims)),
	}
	dScope, ok := toDefineScope(rscope)
	if !ok {
		return ext, false
	}
	for i, dim := range r.dims {
		var dimOk bool
		ext.Ax[i], dimOk = dim.build(dScope)
		ok = ok && dimOk
	}
	return ext, ok
}

func (r *rank) String() string {
	s := strings.Builder{}
	for _, axis := range r.dims {
		s.WriteString(axis.String())
	}
	return s.String()
}

type genericRank struct {
	src *ast.ArrayType
}

var _ rankNode = (*genericRank)(nil)

func (r *genericRank) build(localScope) (ir.ArrayRank, bool) {
	return &ir.RankInfer{Src: r.src}, true
}

func (r *genericRank) String() string {
	return "[...]"
}
