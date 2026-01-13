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
)

type (
	// nodePos is a node attached to some GX source code, pointed by `pos()`
	nodePos interface {
		source() ast.Node
	}

	function interface {
		nodePos
		fnSource() *ast.FuncDecl
		isMethod() bool
		compEval() bool
		resolveOrder() int
		file() *file
		buildSignature(*fileResolveScope) (ir.Func, fnResolveScope, bool)
		buildAnnotations(*fileResolveScope, *irFunc) bool
		buildBody(fnResolveScope, *irFunc) bool
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
		nodePos
		// buildStmt a statement.
		// stop is set to true if the statement always return.
		buildStmt(fnResolveScope) (_ ir.Stmt, stop bool, ok bool)
	}

	cloner interface {
		clone() nodePos
	}
)

func fnName(f function) string {
	return f.fnSource().Name.Name
}

func findStorage(scope resolveScope, name *ast.Ident) (ir.Storage, bool) {
	ns := scope.nspace()
	el, ok := ns.Find(name.Name)
	if !ok {
		return nil, scope.Err().Appendf(name, "undefined: %s", name.Name)
	}
	switch elT := el.(type) {
	case ir.Storage:
		return elT, true
	case ir.WithStore:
		store := elT.Store()
		if store == nil {
			return nil, scope.Err().AppendInternalf(name, "name %q refers element %T which returned a nil storage", name.Name, el)
		}
		return elT.Store(), true
	default:
		return nil, scope.Err().AppendInternalf(name, "element %T is not a storage", el)
	}
}

func storageFromExpr(scope resolveScope, expr ir.Expr) (ir.Storage, bool) {
	withStore, ok := expr.(ir.WithStore)
	if !ok {
		return nil, scope.Err().AppendInternalf(expr.Node(), "%s:%T does not have a store", expr.String(), expr)
	}
	store := withStore.Store()
	if store == nil {
		return nil, false
	}
	return store, true
}

func typeFromStorage(rscope resolveScope, x ir.Expr, store ir.Storage) (*ir.TypeValExpr, bool) {
	tp, ok := store.(ir.Type)
	if ok {
		return ir.TypeExpr(x, tp), true
	}
	value, ok := valueFromStorage(rscope, x, store)
	if !ok {
		return nil, false
	}
	typeRef, ok := value.(*ir.TypeValExpr)
	if !ok {
		return nil, rscope.Err().Appendf(x.Node(), "%s not a type", x.String())
	}
	return typeRef, true
}

func typeFromExpr(rscope resolveScope, x ir.Expr) (*ir.TypeValExpr, bool) {
	store, ok := storageFromExpr(rscope, x)
	if !ok {
		return nil, false
	}
	return typeFromStorage(rscope, x, store)
}

func valueFromStorage(rscope resolveScope, expr ir.Expr, store ir.Storage) (ir.Expr, bool) {
	withValue, ok := store.(ir.StorageWithValue)
	if !ok {
		nameDef := store.NameDef()
		name := "<anonymous>"
		if nameDef != nil {
			name = nameDef.Name
		}
		return nil, rscope.Err().Appendf(store.Node(), "%s undefined", name)
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

var invalidArrayType = ir.NewArrayType(&ast.ArrayType{}, ir.InvalidType(), nil)

func buildArrayType(rscope resolveScope, eNode typeExprNode) (ir.ArrayType, bool) {
	typeExpr, ok := eNode.buildTypeExpr(rscope)
	if !ok {
		return invalidArrayType, false
	}
	arrayType, ok := typeExpr.Val().(ir.ArrayType)
	if !ok {
		return invalidArrayType, rscope.Err().AppendInternalf(eNode.source(), "%s is not an array type", typeExpr.String())
	}
	return arrayType, true
}

var invalidSliceType = &ir.SliceType{
	DType: ir.TypeExpr(nil, ir.InvalidType()),
	Rank:  -1,
}

func buildSliceType(rscope resolveScope, eNode typeExprNode) (*ir.SliceType, bool) {
	typeExpr, ok := eNode.buildTypeExpr(rscope)
	if !ok {
		return invalidSliceType, false
	}
	sliceType, ok := typeExpr.Val().(*ir.SliceType)
	if !ok {
		return invalidSliceType, rscope.Err().AppendInternalf(eNode.source(), "%s is not a slice type", typeExpr.String())
	}
	return sliceType, true
}

func buildCall(rscope resolveScope, eNode exprNode) (ir.CallExpr, bool) {
	expr, ok := eNode.buildExpr(rscope)
	if !ok {
		return nil, false
	}
	call, ok := expr.(ir.CallExpr)
	if !ok {
		return nil, rscope.Err().AppendInternalf(eNode.source(), "cannot call non-function %s (variable of type %s)", expr.String(), expr.Type().String())
	}
	return call, true
}

func assignableToAt(rscope resolveScope, pos ast.Node, src, dst ir.Type) bool {
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return false
	}
	return ir.AssignableToAt(compEval, pos, src, dst)
}

func assignableTo(rscope resolveScope, src, dst ir.Type) (bool, error) {
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return false, nil
	}
	return ir.AssignableTo(compEval, src, dst)
}

func convertToAt(rscope resolveScope, pos ast.Node, src, dst ir.Type) bool {
	convertOk, err := convertTo(rscope, src, dst)
	if err != nil {
		return rscope.Err().AppendAt(pos, err)
	}
	if !convertOk {
		return rscope.Err().Appendf(pos, "cannot convert %s to %s", src.String(), dst.String())
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
	if ir.IsInvalidType(src) || ir.IsInvalidType(dst) {
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
	if ir.IsInvalidType(src) || ir.IsInvalidType(dst) {
		// An error should have already been reported. We skip the check
		// to prevent additional confusing errors.
		return true
	}
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return true
	}
	eq, err := ir.Equal(compEval, src, dst)
	if err != nil {
		rscope.Err().AppendInternalf(pos, "cannot compare types: %v", err)
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
		rscope.Err().AppendInternalf(pos, "cannot compare types: %v", err)
		return true
	}
	return eq
}

func appendRedeclaredError(errF *fmterr.Appender, name string, cur ast.Node, prev ast.Node) bool {
	return errF.Appendf(
		cur,
		"%s redeclared in this block\n\t%s: other declaration of %s",
		name,
		fmterr.At(errF.FSet().FSet, prev),
		name,
	)
}

func isInvalidExpr(expr ir.Expr) bool {
	if expr == nil {
		return true
	}
	return ir.IsInvalidType(expr.Type())
}

var invalidIdent = &ir.Ident{
	Src:  &ast.Ident{Name: "<<<invalid>>>"},
	Stor: ir.InvalidType(),
}

func invalidExpr() *ir.Ident {
	return invalidIdent
}

var invalidGroup = &ir.FieldGroup{Type: ir.TypeExpr(
	invalidExpr(),
	ir.InvalidType(),
)}

func buildAExpr(rscope resolveScope, expr exprNode) (ir.Expr, bool) {
	ext, exprOk := expr.buildExpr(rscope)
	if !exprOk {
		return invalidExpr(), false
	}
	asExt, aexprOk := ext.(ir.Expr)
	if !aexprOk {
		return invalidExpr(), rscope.Err().Appendf(expr.source(), "%s %s is not assignable", ext.String(), ext.Type().Kind().String())
	}
	return asExt, exprOk && !isInvalidExpr(ext)
}
