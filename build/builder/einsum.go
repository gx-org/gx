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
	"fmt"
	"go/ast"
	"go/token"
	"slices"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

type axis struct {
	index int
	name  string
}

type identMap map[string]int

func (s *identMap) add(ident *ast.Ident, index int) {
	(*s)[ident.Name] = index
}

func (s *identMap) contains(ident *ast.Ident) bool {
	_, result := (*s)[ident.Name]
	return result
}

func (s *identMap) difference(other *identMap) *identMap {
	result := identMap{}
	for name, index := range *s {
		if _, ok := (*other)[name]; !ok {
			result[name] = index
		}
	}
	return &result
}

func (s *identMap) intersection(other *identMap) *identMap {
	result := identMap{}
	for name, index := range *s {
		if _, ok := (*other)[name]; ok {
			result[name] = index
		}
	}
	return &result
}

func (s *identMap) orderedAxes() []axis {
	axes := make([]axis, 0, len(*s))
	for name, idx := range *s {
		axes = append(axes, axis{index: idx, name: name})
	}
	slices.SortFunc(axes, func(a, b axis) int {
		return a.index - b.index
	})
	return axes
}

// NOTE: mergeSet yields an ident map with unusable indices; treat its result as a set of names.
func (s *identMap) mergeInto(result *identMap) {
	for name := range *s {
		(*result)[name] = -1
	}
}

type tensorRef struct {
	base     ast.Expr
	exp      exprNode
	indices  []*ast.Ident
	indexMap *identMap
}

func (r *tensorRef) String() string {
	return fmt.Sprintf("%s%s", r.base, r.indices)
}

func (r *tensorRef) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	x, ok := r.exp.buildExpr(rscope)
	if !ok {
		return x, false
	}
	if x.Type().Kind() != irkind.Array {
		return x, rscope.Err().Appendf(r.base, "tensor statements must only reference tensors; %s is %s", r.base, x.Type().String())
	}
	return x, true
}

func (r *tensorRef) source() ast.Node {
	return r.base
}

func processTensorRef(pscope procScope, target *tensorRef, expr ast.Expr, others ...*tensorRef) (*tensorRef, bool) {
	if binExp, ok := expr.(*ast.BinaryExpr); ok {
		// Handle tensor expressions with more than two operands: process the subexpression and convert
		// its result type into a synthetic tensorRef.
		subExp, ok := processTensorExpr(pscope, target, binExp, others...)
		if !ok {
			return nil, false
		}
		return subExp.getResultTensorRef(), true
	}

	var elts []ast.Expr
	if comp, ok := expr.(*ast.CompositeLit); ok {
		expr, elts = comp.Type, comp.Elts
	}
	switch expr.(type) {
	case *ast.Ident:
	case *ast.IndexExpr:
	default:
		pscope.Err().Appendf(expr, "invalid tensor reference base: %T", expr)
		return nil, false
	}

	exp, ok := processExpr(pscope, expr)
	if !ok || exp == nil {
		return nil, false
	}

	var indices []*ast.Ident
	idents := &identMap{}
	for i, elt := range elts {
		ident, ok := elt.(*ast.Ident)
		if !ok {
			pscope.Err().Appendf(elt, "expected tensor reference to index using bare variable, got %T", elt)
			return nil, false
		}
		if idents.contains(ident) {
			pscope.Err().Appendf(elt, "tensor reference includes axis %q more than once", ident)
			return nil, false
		}
		indices = append(indices, ident)
		idents.add(ident, i)
	}
	return &tensorRef{base: expr, exp: exp, indices: indices, indexMap: idents}, true
}

type tensorExpr struct {
	src *ast.BinaryExpr

	target   *tensorRef
	lhs, rhs *tensorRef

	others []*tensorRef
}

var _ exprNode = (*tensorExpr)(nil)

func processEinsumExpr(pscope procScope, left ast.Expr, right *ast.CallExpr) (exprNode, bool) {
	target, ok := processTensorRef(pscope, nil, left)
	if !ok {
		return nil, false
	}
	return processTensorExpr(pscope, target, right.Args[0])
}

func processTensorExpr(pscope procScope, target *tensorRef, expr ast.Expr, others ...*tensorRef) (*tensorExpr, bool) {
	binExp, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return nil, pscope.Err().Appendf(expr, "expected a binary expression, got %T", expr)
	}
	if binExp.Op != token.MUL {
		return nil, pscope.Err().Appendf(expr, "expected a multiply operation, got %q", binExp.Op.String())
	}

	rhs, ok := processTensorRef(pscope, target, binExp.Y)
	if !ok {
		return nil, false
	}
	lhs, ok := processTensorRef(pscope, target, binExp.X, append(append(([]*tensorRef)(nil), others...), rhs)...)
	if !ok {
		return nil, false
	}
	return &tensorExpr{
		src:    binExp,
		target: target,
		others: others,
		lhs:    lhs,
		rhs:    rhs,
	}, true
}

func (s *tensorExpr) buildExpr(scope resolveScope) (ir.Expr, bool) {
	lhs, xOk := s.lhs.buildExpr(scope)
	rhs, yOk := s.rhs.buildExpr(scope)
	ext := &ir.EinsumExpr{Src: s.src, X: lhs, Y: rhs}
	if !xOk || !yOk {
		return ext, false
	}
	lhsTyp, xOk := lhs.Type().(ir.ArrayType)
	if !xOk {
		return ext, scope.Err().AppendInternalf(lhs.Node(), "%s:%s:%T is not an array type", lhs.String(), lhs.Type().String(), lhs.Type())
	}
	rhsTyp, yOk := rhs.Type().(ir.ArrayType)
	if !yOk {
		return ext, scope.Err().AppendInternalf(rhs.Node(), "%s:%s:%T is not an array type", rhs.String(), rhs.Type().String(), rhs.Type())
	}
	leftRank := lhsTyp.Rank()
	rightRank := rhsTyp.Rank()
	targetRank := &ir.Rank{}
	// TODO: Enforce correct output axis ordering: batch dimensions (in LHS order), then LHS cross
	// followed by RHS cross dimensions.
	ext.BatchAxes = findBatchAxes(s.target, s.lhs, s.rhs, s.others...)
	ext.ReduceAxes = findReduceAxes(s.target, s.lhs, s.rhs, s.others...)
	for _, lhsAxis := range ext.BatchAxes[0] {
		targetRank.Ax = append(targetRank.Ax, leftRank.Axes()[lhsAxis])
	}
	crossAxes := findCrossAxes(s.target, s.lhs, s.rhs, s.others...)
	for _, axis := range crossAxes[0] {
		targetRank.Ax = append(targetRank.Ax, leftRank.Axes()[axis])
	}
	for _, axis := range crossAxes[1] {
		targetRank.Ax = append(targetRank.Ax, rightRank.Axes()[axis])
	}
	ext.Typ = ir.NewArrayType(&ast.ArrayType{}, lhsTyp.DataType(), targetRank)
	return ext, true
}

func (s *tensorExpr) getResultTensorRef() *tensorRef {
	i := 0
	var indices []*ast.Ident
	idents := &identMap{}

	batchAxes := findBatchAxes(s.target, s.lhs, s.rhs, s.others...)
	for _, axis := range batchAxes[0] {
		indices = append(indices, s.lhs.indices[axis])
		idents.add(s.lhs.indices[axis], i)
		i++
	}

	crossAxes := findCrossAxes(s.target, s.lhs, s.rhs, s.others...)
	for _, axis := range crossAxes[0] {
		indices = append(indices, s.lhs.indices[axis])
		idents.add(s.lhs.indices[axis], i)
		i++
	}
	for _, axis := range crossAxes[1] {
		indices = append(indices, s.rhs.indices[axis])
		idents.add(s.rhs.indices[axis], i)
		i++
	}
	return &tensorRef{base: s.src, exp: s, indices: indices, indexMap: idents}
}

func (s *tensorExpr) source() ast.Node {
	return s.src
}

func (s *tensorExpr) String() string {
	return "tensor expression"
}

func findBatchAxes(target, lhs, rhs *tensorRef, others ...*tensorRef) [2][]int {
	// Batch axes are present in target, left-hand side, right-hand side, and in every other tensor
	// reference in the statement.
	tmp := rhs.indexMap.intersection(target.indexMap)
	lhsBatch := lhs.indexMap.intersection(tmp) // variables and their indices in lhs
	rhsBatch := tmp.intersection(lhs.indexMap) // variables and their indices in rhs
	for _, other := range others {
		lhsBatch = lhsBatch.intersection(other.indexMap)
		rhsBatch = rhsBatch.intersection(other.indexMap)
	}

	lhsAxes, rhsAxes := []int{}, []int{}
	for _, axis := range lhsBatch.orderedAxes() {
		lhsAxes = append(lhsAxes, axis.index)
		rhsAxes = append(rhsAxes, (*rhsBatch)[axis.name])
	}
	return [2][]int{lhsAxes, rhsAxes}
}

func findReduceAxes(target, lhs, rhs *tensorRef, others ...*tensorRef) [2][]int {
	// Reduction axes appear in both the left-hand side and right-hand side, but nowhere else.
	rhsReduce := rhs.indexMap.difference(target.indexMap)
	lhsReduce := lhs.indexMap.intersection(rhsReduce)
	rhsReduce = rhsReduce.intersection(lhsReduce)
	for _, other := range others {
		lhsReduce = lhsReduce.difference(other.indexMap)
		rhsReduce = rhsReduce.difference(other.indexMap)
	}

	lhsAxes, rhsAxes := []int{}, []int{}
	for _, axis := range lhsReduce.orderedAxes() {
		lhsAxes = append(lhsAxes, axis.index)
		rhsAxes = append(rhsAxes, (*rhsReduce)[axis.name])
	}
	return [2][]int{lhsAxes, rhsAxes}
}

func findCrossAxes(target, lhs, rhs *tensorRef, others ...*tensorRef) [2][]int {
	merged := &identMap{}
	target.indexMap.mergeInto(merged)
	for _, other := range others {
		other.indexMap.mergeInto(merged)
	}

	// Cross product axes appear in exclusively one of the left-hand or right-hand sides, plus in the
	// target and/or another tensor reference in the statement.
	lhsCross := lhs.indexMap.difference(rhs.indexMap).intersection(merged)
	rhsCross := rhs.indexMap.difference(lhs.indexMap).intersection(merged)

	lhsAxes, rhsAxes := []int{}, []int{}
	for _, axis := range lhsCross.orderedAxes() {
		lhsAxes = append(lhsAxes, axis.index)
	}
	for _, axis := range rhsCross.orderedAxes() {
		rhsAxes = append(rhsAxes, axis.index)
	}
	return [2][]int{lhsAxes, rhsAxes}
}

const einsum = "einsum"

// toEinsumCall returns a einsum AST call expression if it is one, or nil otherwise.
func toEinsumCall(expr ast.Expr) *ast.CallExpr {
	call, isCall := expr.(*ast.CallExpr)
	if !isCall {
		return nil
	}
	ident, isIdent := call.Fun.(*ast.Ident)
	if !isIdent {
		return nil
	}
	if ident.Name != einsum {
		return nil
	}
	return call
}
