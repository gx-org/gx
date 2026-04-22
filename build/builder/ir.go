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
	"github.com/gx-org/gx/build/ir/irkind"
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

func typeError(rscope resolveScope, x ir.Expr) (*ir.TypeValExpr, bool) {
	return invalidTypeExprVal, rscope.Err().Appendf(x.Node(), "%s is not a type", x.SourceString(rscope.fileScope().irFile()))
}

func funcError(rscope resolveScope, x ir.Expr) (ir.Expr, bool) {
	return invalidExpr(), rscope.Err().Appendf(x.Node(), "%s is not a function", x.SourceString(rscope.fileScope().irFile()))
}

var invalidArrayType = ir.NewArrayType(&ast.ArrayType{}, ir.InvalidType(), nil)

func buildArrayType(rscope resolveScope, eNode typeExprNode) (ir.ArrayType, bool) {
	typeExpr, ok := eNode.buildTypeExpr(rscope)
	if !ok {
		return invalidArrayType, false
	}
	arrayType, ok := typeExpr.Val().(ir.ArrayType)
	if !ok {
		return invalidArrayType, rscope.Err().AppendInternalf(eNode.source(), "%s is not an array type", typeExpr.SourceString(rscope.fileScope().irFile()))
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
		return invalidSliceType, rscope.Err().AppendInternalf(eNode.source(), "%s is not a slice type", typeExpr.SourceString(rscope.fileScope().irFile()))
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
		from := rscope.fileScope().irFile()
		return nil, rscope.Err().AppendInternalf(eNode.source(), "cannot call non-function %s (variable of type %s)", expr.SourceString(from), expr.Type().ReferString(from))
	}
	return call, true
}

func assignableToAt(rscope resolveScope, pos ast.Node, src, dst ir.Type) bool {
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return false
	}
	assignable, cpErr, err := ir.AssignableTo(compEval, src, dst)
	if err != nil {
		return compEval.Err().AppendAt(pos, err)
	}
	if cpErr != nil {
		return compEval.Err().AppendAt(pos, cpErr)
	}
	if !assignable {
		return compEval.Err().Appendf(pos, "cannot use %s as %s value in assignment", src.ReferString(compEval.File()), dst.ReferString(compEval.File()))
	}
	return true
}

func assignableTo(rscope resolveScope, src, dst ir.Type) (bool, ir.CompEvalError, error) {
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return false, nil, nil
	}
	return ir.AssignableTo(compEval, src, dst)
}

func convertToAt(rscope resolveScope, pos ast.Node, src, dst ir.Type) bool {
	convertOk, cpErr, err := convertTo(rscope, src, dst)
	if err != nil {
		return rscope.Err().AppendAt(pos, err)
	}
	if cpErr != nil {
		return rscope.Err().AppendAt(pos, cpErr)
	}
	if !convertOk {
		from := rscope.fileScope().irFile()
		return rscope.Err().Appendf(pos, "cannot convert %s to %s", src.ReferString(from), dst.ReferString(from))
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

func convertTo(rscope resolveScope, src, dst ir.Type) (bool, ir.CompEvalError, error) {
	if ir.IsInvalidType(src) || ir.IsInvalidType(dst) {
		// An error should have already been reported. We skip the check
		// to prevent additional confusing errors.
		return true, nil, nil
	}
	reconcile(src, dst)
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return true, nil, nil
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
	eq, cpErr, err := ir.Equal(compEval, src, dst)
	if err != nil {
		rscope.Err().AppendInternalf(pos, "cannot compare types: %v", err)
		return true
	}
	if cpErr != nil {
		return rscope.Err().AppendAt(pos, cpErr)
	}
	return eq
}

func axisEqual(rscope resolveScope, pos ast.Node, src, dst ir.AxisLengths) bool {
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return true
	}
	eq, cpErr, err := src.Equal(compEval, dst)
	if err != nil {
		rscope.Err().AppendInternalf(pos, "cannot compare types: %v", err)
		return true
	}
	if cpErr != nil {
		return rscope.Err().AppendAt(pos, cpErr)
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

var invalidTypeExprVal = ir.TypeExpr(
	invalidExpr(),
	ir.InvalidType(),
)

var invalidGroup = &ir.FieldGroup{
	Type: invalidTypeExprVal,
}

func buildCoreExpr(rscope resolveScope, expr exprNode) (ir.Expr, bool) {
	ext, exprOk := expr.buildExpr(rscope)
	if !exprOk {
		return invalidExpr(), false
	}
	asExt, aexprOk := ext.(ir.Expr)
	if !aexprOk {
		return invalidExpr(), rscope.Err().Appendf(expr.source(), "%s %s is not assignable", ext.SourceString(rscope.fileScope().irFile()), ext.Type().Kind().String())
	}
	return asExt, exprOk && !isInvalidExpr(ext)
}

func buildExpr(rscope resolveScope, expr exprNode) (ir.Expr, bool) {
	ext, ok := buildCoreExpr(rscope, expr)
	if !ok {
		return ext, false
	}
	knd := ext.Type().Kind()
	if knd != irkind.Invalid && knd == irkind.MetaType {
		return ext, rscope.Err().Appendf(expr.source(), "%s (type) is not an expression", ext.SourceString(rscope.fileScope().irFile()))
	}
	return ext, true
}

func castNilAndNumber(scope resolveScope, expr ir.Expr, target ir.Type) (ir.Expr, bool) {
	if isInvalidExpr(expr) || !ir.IsValid(target) {
		return expr, false
	}
	knd := expr.Type().Kind()
	if irkind.IsNumber(knd) {
		return ir.CastNumber(toFileWithError(scope), expr, target)
	}
	if knd == irkind.Nil {
		return castNil(scope, expr, target)
	}
	return expr, !isInvalidExpr(expr) && ir.IsValid(target)
}

type fileWithError struct {
	resolveScope
}

func toFileWithError(rscope resolveScope) ir.FileWithError {
	return &fileWithError{resolveScope: rscope}
}
func (fwe *fileWithError) File() *ir.File {
	return fwe.resolveScope.fileScope().irFile()
}
