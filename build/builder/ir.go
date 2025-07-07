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

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
)

type (
	// nodePos is a node attached to some GX source code, pointed by `pos()`
	nodePos interface {
		source() ast.Node
	}

	function interface {
		nodePos
		name() *ast.Ident
		isMethod() bool
		compEval() bool
		resolveOrder() int
		buildSignature(*pkgResolveScope) (ir.Func, iFuncResolveScope, bool)
		buildBody(iFuncResolveScope, ir.Func) ([]*cpevelements.SyntheticFuncDecl, bool)
	}

	// exprNode builds a IR expression.
	exprNode interface {
		nodePos

		buildExpr(resolveScope) (ir.Expr, bool)

		String() string
	}

	typeExprNode interface {
		nodePos

		buildTypeExpr(resolveScope) (*ir.TypeValExpr, bool)

		String() string
	}

	// stmtNode is a GX statement.
	stmtNode interface {
		buildStmt(iFuncResolveScope) (ir.Stmt, bool)
	}

	cloner interface {
		clone() nodePos
	}
)

func storageFromExpr(scope resolveScope, expr ir.Expr) (ir.Storage, bool) {
	withStore, ok := expr.(ir.WithStore)
	if !ok {
		return nil, scope.err().AppendInternalf(expr.Source(), "%T does not have a store", expr)
	}
	store := withStore.Store()
	if store == nil {
		return nil, false
	}
	return store, true
}

func typeFromStorage(rscope resolveScope, x ir.AssignableExpr, store ir.Storage) (*ir.TypeValExpr, bool) {
	value, ok := valueFromStorage(rscope, x, store)
	if !ok {
		return nil, false
	}
	typeRef, ok := value.(*ir.TypeValExpr)
	if !ok {
		return nil, rscope.err().Appendf(x.Source(), "%s not a type", x.String())
	}
	return &ir.TypeValExpr{X: x, Typ: typeRef.Typ}, true
}

func typeFromExpr(rscope resolveScope, x ir.AssignableExpr) (*ir.TypeValExpr, bool) {
	store, ok := storageFromExpr(rscope, x)
	if !ok {
		return nil, false
	}
	return typeFromStorage(rscope, x, store)
}

func valueFromStorage(rscope resolveScope, expr ir.Expr, store ir.Storage) (ir.AssignableExpr, bool) {
	withValue, ok := store.(ir.StorageWithValue)
	if !ok {
		nameDef := store.NameDef()
		name := "<anonymous>"
		if nameDef != nil {
			name = nameDef.Name
		}
		return nil, rscope.err().AppendInternalf(store.Source(), "storage %T:%s:%s has no value", store, name, store.Type().String())
	}
	return withValue.Value(expr), true
}

func buildTypeExpr(rscope resolveScope, node exprNode) (*ir.TypeValExpr, bool) {
	typExpr, ok := node.(typeExprNode)
	if ok {
		return typExpr.buildTypeExpr(rscope)
	}
	expr, ok := buildAExpr(rscope, node)
	if !ok {
		return nil, false
	}
	return typeFromExpr(rscope, expr)
}

var invalidArrayType = ir.NewArrayType(nil, ir.InvalidType(), nil)

func buildArrayType(rscope resolveScope, eNode typeExprNode) (ir.ArrayType, bool) {
	typeExpr, ok := eNode.buildTypeExpr(rscope)
	if !ok {
		return invalidArrayType, false
	}
	arrayType, ok := typeExpr.Typ.(ir.ArrayType)
	if !ok {
		return invalidArrayType, rscope.err().AppendInternalf(eNode.source(), "%s is not an array type", typeExpr.String())
	}
	return arrayType, true
}

var invalidSliceType = &ir.SliceType{
	DType: &ir.TypeValExpr{Typ: ir.InvalidType()},
	Rank:  -1,
}

func buildSliceType(rscope resolveScope, eNode typeExprNode) (*ir.SliceType, bool) {
	typeExpr, ok := eNode.buildTypeExpr(rscope)
	if !ok {
		return invalidSliceType, false
	}
	sliceType, ok := typeExpr.Typ.(*ir.SliceType)
	if !ok {
		return invalidSliceType, rscope.err().AppendInternalf(eNode.source(), "%s is not a slice type", typeExpr.String())
	}
	return sliceType, true
}

func buildCall(rscope resolveScope, eNode exprNode) (*ir.CallExpr, bool) {
	expr, ok := eNode.buildExpr(rscope)
	if !ok {
		return nil, false
	}
	call, ok := expr.(*ir.CallExpr)
	if !ok {
		return nil, rscope.err().AppendInternalf(eNode.source(), "cannot call non-function %s (variable of type %s)", expr.String(), expr.Type().String())
	}
	return call, true
}

func assignableToAt(rscope resolveScope, pos ast.Node, src, dst ir.Type) bool {
	assignable, err := assignableTo(rscope, src, dst)
	if err != nil {
		return rscope.err().AppendAt(pos, err)
	}
	if !assignable {
		return rscope.err().Appendf(pos, "cannot use %s as %s value in assignment", src.String(), dst.String())
	}
	return true
}

func assignableTo(rscope resolveScope, src, dst ir.Type) (bool, error) {
	if isInvalid(src) && isInvalid(dst) {
		// An error should have already been reported. We skip the check
		// to prevent additional confusing errors.
		return true, nil
	}
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return true, nil
	}
	return src.AssignableTo(compEval, dst)
}

func convertToAt(rscope resolveScope, pos ast.Node, src, dst ir.Type) bool {
	convertOk, err := convertTo(rscope, src, dst)
	if err != nil {
		return rscope.err().AppendAt(pos, err)
	}
	if !convertOk {
		return rscope.err().Appendf(pos, "cannot convert %s to %s", src.String(), dst.String())
	}
	return true
}

func reconcile(src, dst ir.Type) {
	srcArray, srcOk := src.(ir.ArrayType)
	dstArray, dstOk := dst.(ir.ArrayType)
	if !srcOk || !dstOk {
		return
	}
	dstInfer, ok := dstArray.Rank().(*ir.RankInfer)
	if !ok {
		return
	}
	dstInfer.Rnk = srcArray.Rank()
}

func convertTo(rscope resolveScope, src, dst ir.Type) (bool, error) {
	if isInvalid(src) && isInvalid(dst) {
		// An error should have already been reported. We skip the check
		// to prevent additional confusing errors.
		return true, nil
	}
	reconcile(src, dst)
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return true, nil
	}
	return src.ConvertibleTo(compEval, dst)
}

func equalToAt(rscope resolveScope, pos ast.Node, src, dst ir.Type) bool {
	if isInvalid(src) && isInvalid(dst) {
		// An error should have already been reported. We skip the check
		// to prevent additional confusing errors.
		return true
	}
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return true
	}
	eq, err := src.Equal(compEval, dst)
	if err != nil {
		rscope.err().AppendInternalf(pos, "cannot compare types: %v", err)
		return true
	}
	return eq
}

func axisEqual(rscope resolveScope, pos ast.Node, src, dst ir.AxisLengths) bool {
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return true
	}
	eq, err := src.Equal(compEval, dst)
	if err != nil {
		rscope.err().AppendInternalf(pos, "cannot compare types: %v", err)
		return true
	}
	return eq
}

func appendRedeclaredError(errF *fmterr.Appender, name string, cur ast.Node, prev ast.Node) bool {
	return errF.Appendf(
		cur,
		"%s redeclared in this block\n\t%s: other declaration of %s",
		name,
		fmterr.PosString(errF.FSet().FSet, prev.Pos()),
		name,
	)
}

func isInvalid(typ ir.Type) bool {
	return typ == nil || typ.Kind() == ir.InvalidKind
}

func buildAExpr(rscope resolveScope, expr exprNode) (ir.AssignableExpr, bool) {
	ext, ok := expr.buildExpr(rscope)
	if !ok {
		return nil, false
	}
	asExt, ok := ext.(ir.AssignableExpr)
	if !ok {
		return nil, rscope.err().Appendf(expr.source(), "%s %s is not assignable", ext.String(), ext.Type().Kind().String())
	}
	return asExt, true
}
