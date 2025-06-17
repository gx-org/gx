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
	"reflect"
	"strings"

	"github.com/gx-org/gx/build/ir/generics"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
)

type (
	rankableType interface {
		rank() rankNode
	}

	// indexExpr references an element (or subarray) of an array.
	indexExpr struct {
		src *ast.IndexExpr

		x     exprNode
		index exprNode
	}
)

var _ exprNode = (*indexExpr)(nil)

func processIndexExpr(pscope procScope, src *ast.IndexExpr) (*indexExpr, bool) {
	n := &indexExpr{src: src}
	var xOk, indexOk bool
	n.x, xOk = processExpr(pscope, src.X)
	n.index, indexOk = processExpr(pscope, src.Index)
	return n, xOk && indexOk
}

func (n *indexExpr) source() ast.Node {
	return n.src
}

func (n *indexExpr) checkIndexType(scope *fileResolveScope, index ir.Expr) (ir.Expr, bool) {
	var ok bool
	if ir.IsNumber(index.Type().Kind()) {
		// Coerce index type to a concrete integer type.
		index, ok = castNumber(scope, index, ir.DefaultIntType)
	}
	if !ok {
		return index, false
	}
	if !ir.IsIndexType(index.Type()) {
		return index, scope.err().Appendf(n.source(), "invalid argument: index %s must be integer", index.Type().String())
	}
	return index, true
}

func (n *indexExpr) checkIndexBounds(rscope resolveScope, axLen ir.AxisLengths, index ir.Expr) bool {
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return false
	}
	axisValue, err := compEval.Eval(axLen)
	if err != nil {
		return rscope.err().AppendInternalf(axLen.Source(), "cannot evaluate axis length expression: %v", err)
	}
	indexValue, err := compEval.Eval(index)
	if err != nil {
		return rscope.err().AppendInternalf(axLen.Source(), "cannot evaluate slice index expression: %v", err)
	}
	axisInt := canonical.ToValue(axisValue)
	indexInt := canonical.ToValue(indexValue)
	if axisInt == nil || indexInt == nil {
		return true
	}
	if indexInt.Cmp(axisInt) >= 0 {
		return rscope.err().Appendf(n.source(), "index out of range: %s >= %s", indexInt.String(), axisInt.String())
	}
	return true
}

func specializeFunc(rscope resolveScope, x ir.Expr, indices []ir.AssignableExpr) (ir.Expr, bool) {
	funStore, ok := storageFromExpr(rscope, x)
	if !ok {
		return nil, false
	}
	funValue, ok := valueFromStorage(rscope, x, funStore)
	if !ok {
		return nil, false
	}
	fun, ok := funValue.(*ir.FuncValExpr)
	if !ok {
		return x, rscope.err().AppendInternalf(x.Source(), "%s is not a function: %T", x, x)
	}
	typeExprs := make([]*ir.TypeValExpr, len(indices))
	indicesOk := true
	for i, index := range indices {
		var iOk bool
		typeExprs[i], iOk = typeFromExpr(rscope, index)
		indicesOk = indicesOk && iOk
	}
	if !ok || !indicesOk {
		return x, false
	}
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return x, false
	}

	specFunType, ok := generics.Specialise(compEval, x, fun, typeExprs)
	if !ok {
		return x, false
	}
	return specFunType.Value(x), ok
}

func (n *indexExpr) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	x, xOk := n.x.buildExpr(rscope)
	idx, idxOk := buildAExpr(rscope, n.index)
	if !xOk || !idxOk {
		return nil, false
	}
	xType := x.Type()
	if xType.Kind() == ir.FuncKind {
		return specializeFunc(rscope, x, []ir.AssignableExpr{idx})
	}
	ext := &ir.IndexExpr{Src: n.src, X: x, Index: idx, Typ: ir.InvalidType()}
	if !ir.IsSlicingOk(xType) {
		return ext, rscope.err().Appendf(n.source(), "cannot index %s (type: %s)", ext.X, xType)
	}
	slicerType, ok := ir.Underlying(xType).(ir.SlicerType)
	if !ok {
		return ext, rscope.err().AppendInternalf(n.source(), "type %T with kind %s supports slicing but does not implement %s", xType, xType.Kind().String(), reflect.TypeFor[ir.SlicerType]())
	}
	ext.Typ, ok = slicerType.ElementType()
	if !ok {
		return ext, rscope.err().Appendf(n.source(), "cannot index %s", xType)
	}
	aType, isArray := xType.(ir.ArrayType)
	boundOk := true
	if isArray {
		boundOk = n.checkIndexBounds(rscope, aType.Rank().Axes()[0], ext.Index)

	}
	numberOk := true
	if ir.IsNumber(ext.Index.Type().Kind()) {
		ext.Index, numberOk = castNumber(rscope, ext.Index, ir.Int64Type())
	}
	return ext, boundOk && numberOk
}

func (n *indexExpr) String() string {
	return fmt.Sprintf("%s[%s]", n.x.String(), n.index.String())
}

// indexExpr references an element (or subarray) of an array.
type indexListExpr struct {
	src *ast.IndexListExpr

	x       exprNode
	indices []exprNode
}

var _ exprNode = (*indexListExpr)(nil)

func processIndexListExpr(pscope procScope, src *ast.IndexListExpr) (*indexListExpr, bool) {
	n := &indexListExpr{src: src, indices: make([]exprNode, len(src.Indices))}
	var xOk bool
	n.x, xOk = processExpr(pscope, src.X)
	indicesOk := true
	for i, index := range src.Indices {
		var iOk bool
		n.indices[i], iOk = processExpr(pscope, index)
		indicesOk = iOk && indicesOk
	}
	return n, xOk && indicesOk
}

func (n *indexListExpr) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	x, xOk := n.x.buildExpr(rscope)
	indices := make([]ir.AssignableExpr, len(n.indices))
	indicesOk := true
	for i, index := range n.indices {
		var iOk bool
		indices[i], iOk = buildAExpr(rscope, index)
		indicesOk = indicesOk && iOk
	}
	if !xOk || !indicesOk {
		return x, false
	}
	if x.Type().Kind() != ir.FuncKind {
		return x, rscope.err().Appendf(n.source(), "list of indices only supported for generic functions")
	}
	return specializeFunc(rscope, x, indices)
}

func (n *indexListExpr) source() ast.Node {
	return n.src
}

func (n *indexListExpr) String() string {
	indices := make([]string, len(n.indices))
	for i, index := range n.indices {
		indices[i] = index.String()
	}
	return fmt.Sprintf("%s[%s]", n.x.String(), strings.Join(indices, ","))
}
