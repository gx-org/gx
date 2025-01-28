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
	"strings"

	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/gx/build/ir"
)

// arrayType defines an array type defined in the source.
type arrayType struct {
	ast  *ast.ArrayType
	dtyp typeNode
	rnk  rankNode
}

var (
	_ concreteTypeNode = (*arrayType)(nil)
	_ indexableType    = (*arrayType)(nil)
	_ arrayTypeNode    = (*arrayType)(nil)
)

func processArrayType(owner owner, typ *ast.ArrayType) (*arrayType, bool) {
	rank, dtype, ok := processDTypeRank(owner, typ)
	return &arrayType{
		ast:  typ,
		dtyp: dtype,
		rnk:  rank,
	}, ok
}

func importArrayType(scope scoper, ext ir.ArrayType) (*arrayType, bool) {
	typ := &arrayType{
		ast: ext.ArrayType(),
	}
	var rankOk bool
	typ.rnk, rankOk = importRank(scope, ext.Rank())
	var dtypeOk bool
	typ.dtyp, dtypeOk = toTypeNode(scope, ext.DataType())
	return typ, rankOk && dtypeOk
}

func (n *arrayType) elementType() (typeNode, error) {
	return &arrayType{
		ast:  n.ast,
		rnk:  n.rnk.subRank(),
		dtyp: n.dtyp,
	}, nil
}

func (n *arrayType) source() ast.Node {
	return n.ast
}

func (n *arrayType) irType() ir.Type {
	return n.buildArrayType()
}

func (n *arrayType) buildArrayType() ir.ArrayType {
	return ir.NewArrayType(n.ast, n.dtyp.irType(), n.rnk.build())
}

func (n *arrayType) kind() ir.Kind {
	if n.rnk.numAxes() == 0 {
		return n.dtyp.kind()
	}
	if n.dtyp.kind() == ir.InvalidKind {
		return ir.InvalidKind
	}
	return ir.ArrayKind
}

func (n *arrayType) rank() rankNode {
	return n.rnk
}

func (n *arrayType) dtype() typeNode {
	return n.dtyp
}

func (n *arrayType) reconcileDType(scope scoper, pos nodePos, other arrayTypeNode) (typeNode, bool) {
	dtype := n.dtyp
	if dtype.kind() == ir.UnknownKind {
		return other.dtype(), true
	}
	eq, err := dtype.irType().Equal(scope.evalFetcher(), other.dtype().irType())
	if err != nil {
		scope.err().Append(scope.err().FSet().Position(pos.source(), err))
		return n.dtyp, false
	}
	if !eq {
		return n.dtyp, scope.err().Appendf(pos.source(), "cannot reconcile %s with %s", dtype.String(), other.String())
	}
	return dtype, eq
}

func (n *arrayType) reconcileWith(scope scoper, pos nodePos, otherN typeNode) (typeNode, bool) {
	other, ok := otherN.(arrayTypeNode)
	if !ok {
		scope.err().Appendf(n.source(), "cannot reconcile an array type with %T", other)
		return n, false
	}
	dtype, dtypeOk := n.reconcileDType(scope, pos, other)
	rank, rankOk := n.rnk.reconcileWith(scope, pos, other.rank())
	return &arrayType{
		ast:  n.ast,
		dtyp: dtype,
		rnk:  rank,
	}, dtypeOk && rankOk
}

func (n *arrayType) convertTo(scope scoper, pos nodePos, dstN typeNode) (typeNode, bool) {
	dst, ok := dstN.(arrayTypeNode)
	if !ok {
		return invalid, scope.err().AppendInternalf(pos.source(), "cannot convert array to type %T", dst)
	}
	dstRank := dst.rank()
	genRank, ok := dst.rank().(*genericRank)
	if ok && genRank.rnk == nil {
		dstRank = n.rank()
	}
	return &arrayType{
		ast:  n.ast,
		dtyp: dst.dtype(),
		rnk:  dstRank,
	}, true
}

func (n *arrayType) isGeneric() bool {
	return n.rank().build().IsGeneric()
}

func (n *arrayType) lengths(scope scoper, call *ir.CallExpr) (*sliceExpr, bool) {
	rank, ok := n.rnk.(*rank)
	if !ok {
		scope.err().Appendf(call.Src, "cannot get the length of an array with an unknown number of axis")
		return nil, false
	}
	exprs := make([]exprNode, len(rank.dims))
	for i, axis := range rank.dims {
		exprs[i] = axis
	}
	return toSliceExpr(scope, call.Expr(), axisLengthType, exprs), true
}

func (n *arrayType) String() string {
	return n.rnk.String() + n.dtyp.String()
}

func (n *arrayType) resolveConcreteType(scope scoper) (typeNode, bool) {
	dtyp, ok := resolveType(scope, n, n.dtyp)
	if !ok {
		return typeNodeOk(invalid)
	}
	if !ir.IsDataType(dtyp.kind()) {
		scope.err().Appendf(n.source(), "array of %s not supported", dtyp.String())
		return typeNodeOk(invalid)
	}
	// Note that rank.resolve() may return a new concrete rankNode instance if the rank was generic.
	// In that case, we return a new instance of arrayType (see below) and keep the generic arrayType
	// and rank unchanged so they can be specialized again for the next call.
	rank, rankOk := n.rnk.resolve(scope)
	if !rankOk {
		return typeNodeOk(invalid)
	}
	if dtyp != n.dtyp || rank != n.rank() {
		return &arrayType{
			ast:  n.ast,
			dtyp: dtyp,
			rnk:  rank,
		}, true
	}
	return n, true
}

type (
	arrayLiteral interface {
		source() ast.Node
		dtype() ir.Type
		buildExpr(ir.ArrayType) ir.Expr
		String() string
	}

	// tImplicitLiteral parses a tensor with implicit value declaration.
	// For example: [2][3]float32{}
	tImplicitLiteral[T dtype.GoDataType] struct {
		ext       *ir.ArrayLitExpr
		wantDType ir.Type
	}
)

func newImplicitLiteral[T dtype.GoDataType](n *arrayLitExpr) arrayLiteral {
	return &tImplicitLiteral[T]{
		ext:       &ir.ArrayLitExpr{Src: n.src},
		wantDType: n.typ.dtype().irType(),
	}
}

func (arr *tImplicitLiteral[T]) source() ast.Node {
	return arr.ext.Source()
}

func (arr *tImplicitLiteral[T]) dtype() ir.Type {
	return arr.wantDType
}

func (arr *tImplicitLiteral[T]) buildExpr(tp ir.ArrayType) ir.Expr {
	arr.ext.Typ = tp
	return arr.ext
}

func (arr *tImplicitLiteral[T]) String() string {
	return "{}"
}

type (
	// explicitLiteral parses tensor value declaration of the form:
	// [2][3]float32{{1, 2, 3},{4, 5, 6}}
	// where all the values are defined explicitly.
	explicitLiteral interface {
		arrayLiteral

		addAxe(expr exprNode, dim int) explicitLiteral
		appendValue(scoper, exprNode) bool
		appendVals(a explicitLiteral)
		rank() *rank
	}

	tExplicitLiteral[T dtype.GoDataType] struct {
		ext    *ir.ArrayLitExpr
		dtypeF typeNode
		dims   []*inferredFromLiteralAxisLength
	}
)

func newExplicitLiteral[T dtype.GoDataType](n *arrayLitExpr) explicitLiteral {
	return &tExplicitLiteral[T]{
		ext:    &ir.ArrayLitExpr{Src: n.src},
		dtypeF: n.typ.dtyp,
	}
}

func (arr *tExplicitLiteral[T]) appendVals(other explicitLiteral) {
	otherVals := other.(*tExplicitLiteral[T]).ext.Vals
	arr.ext.Vals = append(arr.ext.Vals, otherVals...)
}

func (arr *tExplicitLiteral[T]) buildExpr(typ ir.ArrayType) ir.Expr {
	arr.ext.Typ = typ
	return arr.ext
}

func (arr *tExplicitLiteral[T]) dtype() ir.Type {
	return arr.dtypeF.irType()
}

func (arr *tExplicitLiteral[T]) source() ast.Node {
	return arr.ext.Source()
}

func (arr *tExplicitLiteral[T]) rank() *rank {
	rnk := &rank{
		dims: make([]axisLengthNode, len(arr.dims)),
	}
	for i, dim := range arr.dims {
		rnk.dims[i] = dim
	}
	return rnk
}

func (arr *tExplicitLiteral[T]) appendValue(scope scoper, expr exprNode) bool {
	exprType, ok := expr.resolveType(scope)
	if !ok {
		return false
	}
	if ir.IsNumber(exprType.kind()) {
		if expr, exprType, ok = castNumber(scope, expr, arr.dtypeF.irType()); !ok {
			return false
		}
	}
	var canAssign bool
	var err error
	arr.dtypeF, canAssign, err = assignableTo(scope, arr, exprType, arr.dtypeF)
	if err != nil {
		scope.err().Append(err)
		return false
	}
	if !canAssign {
		scope.err().Appendf(expr.source(), "cannot use value of type %v as value of type %v in array", exprType, arr.dtype())
		return false
	}
	arr.ext.Vals = append(arr.ext.Vals, expr.buildExpr())
	if len(arr.dims) == 0 {
		arr.dims = []*inferredFromLiteralAxisLength{&inferredFromLiteralAxisLength{src: arr.ext.Src}}
	}
	arr.dims[0].val++
	return true
}

func (arr *tExplicitLiteral[T]) addAxe(expr exprNode, dim int) explicitLiteral {
	ldim := &inferredFromLiteralAxisLength{src: arr.ext.Src, val: dim}
	return &tExplicitLiteral[T]{
		ext: &ir.ArrayLitExpr{
			Src:  arr.ext.Src,
			Vals: append([]ir.Expr{}, arr.ext.Vals...),
		},
		dtypeF: arr.dtypeF,
		dims:   append([]*inferredFromLiteralAxisLength{ldim}, arr.dims...),
	}
}

func (arr *tExplicitLiteral[T]) String() string {
	return fmt.Sprintf("%s{#%d}", arr.dtypeF.String(), len(arr.ext.Vals))
}

// arrayLitExpr defines an array with a constant value.
type arrayLitExpr struct {
	src   *ast.CompositeLit
	typ   *arrayType
	array arrayLiteral
}

func (n *arrayType) toExprNode(owner owner, expr *ast.CompositeLit) (exprNode, bool) {
	return &arrayLitExpr{
		typ: n,
		src: expr,
	}, true
}

// Pos returns the position of the array in the code.
func (n *arrayLitExpr) source() ast.Node {
	return n.src
}

func (n *arrayLitExpr) newImplicitLiteral(scope scoper) arrayLiteral {
	kind := n.typ.dtype().kind()
	switch kind {
	case ir.BoolKind:
		return newImplicitLiteral[bool](n)
	case ir.Int32Kind:
		return newImplicitLiteral[int32](n)
	case ir.Int64Kind:
		return newImplicitLiteral[int64](n)
	case ir.Uint32Kind:
		return newImplicitLiteral[uint32](n)
	case ir.Uint64Kind:
		return newImplicitLiteral[uint64](n)
	case ir.Float32Kind:
		return newImplicitLiteral[float32](n)
	case ir.Float64Kind:
		return newImplicitLiteral[float64](n)
	case ir.InvalidKind:
		return nil
	default:
		scope.err().Appendf(n.source(), "kind %s not supported as a data type", kind)
		return nil
	}
}

func (n *arrayLitExpr) newExplicitLiteral(scope scoper) explicitLiteral {
	kind := n.typ.dtype().kind()
	switch kind {
	case ir.BoolKind:
		return newExplicitLiteral[bool](n)
	case ir.Int32Kind:
		return newExplicitLiteral[int32](n)
	case ir.Int64Kind:
		return newExplicitLiteral[int64](n)
	case ir.Uint32Kind:
		return newExplicitLiteral[uint32](n)
	case ir.Uint64Kind:
		return newExplicitLiteral[uint64](n)
	case ir.Float32Kind:
		return newExplicitLiteral[float32](n)
	case ir.Float64Kind:
		return newExplicitLiteral[float64](n)
	case ir.InvalidKind:
		return nil
	default:
		scope.err().Appendf(n.source(), "kind %s not supported as a data type", kind)
		return nil
	}
}

func (n *arrayLitExpr) expr() ast.Expr {
	return n.src
}

func (n *arrayLitExpr) buildExpr() ir.Expr {
	if n.array == nil {
		return nil
	}
	return n.array.buildExpr(n.typ.buildArrayType())
}

func (n *arrayLitExpr) parseValues(scope scoper) explicitLiteral {
	arr := n.newExplicitLiteral(scope)
	if arr == nil {
		return nil
	}
	for _, expr := range n.src.Elts {
		valueExpr, ok := processExpr(scope, expr)
		if !ok {
			return nil
		}
		ok = arr.appendValue(scope, valueExpr)
		if !ok {
			return nil
		}
	}
	return arr
}

func formatDims(dims []axisLengthNode) string {
	s := strings.Builder{}
	for _, dim := range dims {
		s.WriteString(fmt.Sprintf("[%v]", dim))
	}
	return s.String()
}

func (n *arrayLitExpr) parseComposite(scope scoper, expr *ast.CompositeLit) explicitLiteral {
	typ, ok := expr.Type.(*ast.ArrayType)
	if expr.Type != nil && !ok {
		scope.err().Appendf(expr, "cannot use %T as array or slice literal", expr.Type)
		return nil
	}
	var subArrayType *arrayType
	if typ == nil {
		elementType, err := n.typ.elementType()
		if err != nil {
			scope.err().Append(scope.err().FSet().Position(expr, err))
			return nil
		}
		subArrayType, ok = elementType.(*arrayType)
		if !ok {
			scope.err().Appendf(expr, "b-array type is not an array type")
		}
	} else {
		subArrayType, ok = processArrayType(scope, typ)
	}
	if !ok {
		return nil
	}
	if _, ok := resolveType(scope, n, subArrayType); !ok {
		return nil
	}
	subArray := (&arrayLitExpr{typ: subArrayType, src: expr}).parseTensor(scope)
	if subArray == nil {
		return nil
	}
	dtypeGot := subArray.dtype().Kind()
	dtypeWant := n.typ.dtype().kind()
	if dtypeGot != dtypeWant {
		scope.err().Appendf(expr, "cannot use data type %s within a %s array", dtypeGot.String(), dtypeWant.String())
		return nil
	}
	return subArray
}

func (n *arrayLitExpr) parseTensor(scope scoper) (res explicitLiteral) {
	rank := n.typ.rnk
	defer func() {
		if res == nil {
			return
		}
		var ok bool
		n.typ.rnk, ok = rank.reconcileWith(scope, n, res.rank())
		if !ok {
			res = nil
		}
	}()
	lit := n.src
	if rank.numAxes() == 1 || len(lit.Elts) == 0 {
		return n.parseValues(scope)
	}
	if _, ok := lit.Elts[0].(*ast.CompositeLit); !ok {
		return n.parseValues(scope)
	}
	arrays := make([]explicitLiteral, len(lit.Elts))
	for i, expr := range lit.Elts {
		exprT, ok := expr.(*ast.CompositeLit)
		if !ok {
			scope.err().Appendf(expr, "unexpected %T, expected type expression", exprT)
			return nil
		}
		arrays[i] = n.parseComposite(scope, exprT)
		if arrays[i] == nil {
			return nil
		}
	}
	first := arrays[0]
	wantDims := first.rank().dims
	res = first.addAxe(n, len(arrays))
	for _, arr := range arrays[1:] {
		gotDims := arr.rank().dims
		if !lengthEquals(scope, gotDims, wantDims) {
			return nil
		}
		res.appendVals(arr)
	}
	return res
}

// Type returns the type of the tensor.
func (n *arrayLitExpr) resolveType(scope scoper) (typeNode, bool) {
	if _, ok := resolveType(scope, n, n.typ); !ok {
		return n.typ, false
	}
	if len(n.src.Elts) == 0 {
		n.array = n.newImplicitLiteral(scope)
	} else {
		n.array = n.parseTensor(scope)
	}
	if n.array == nil {
		return n.typ, false
	}
	return typeNodeOk(n.typ)
}

func (n *arrayLitExpr) String() string {
	if n.array != nil {
		return n.array.String()
	}
	return n.typ.String() + "{invalid}"
}
