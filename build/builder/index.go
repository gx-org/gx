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
	"strings"

	"github.com/gx-org/gx/build/ir/generics"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/constants"
	"github.com/gx-org/gx/src/gx/ast"
)

// indexExpr references an element (or subarray) of an array.
type indexExpr struct {
	src *ast.IndexExpr

	x     exprNode
	index exprNode
}

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
	// Coerce index type to a concrete integer type.
	index, ok = castNilAndNumber(scope, index, ir.DefaultIntType)
	if !ok {
		return index, false
	}
	if !ir.IsIndexType(index.Type()) {
		return index, scope.Err().Appendf(n.source(), "invalid argument: index %s must be integer", index.Type().ReferString(scope.irFile()))
	}
	return index, true
}

func (n *indexExpr) checkIndexBounds(rscope resolveScope, axLen ir.AxisLengths, index ir.Expr) bool {
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return false
	}
	axisValue, err := compEval.EvalExpr(axLen.AsExpr())
	if err != nil {
		return rscope.Err().AppendInternalf(axLen.Node(), "cannot evaluate axis length expression: %v", err)
	}
	indexValue, err := compEval.EvalExpr(index)
	if err != nil {
		return rscope.Err().AppendInternalf(axLen.Node(), "cannot evaluate slice index expression: %v", err)
	}
	axisInt, axisOk := constants.ConvertOk(constants.CInt, axisValue)
	indexInt, indexOk := constants.ConvertOk(constants.CInt, indexValue)
	if !axisOk || !indexOk {
		return true
	}
	if indexInt >= axisInt {
		return rscope.Err().Appendf(n.source(), "index out of range: %d >= %d", indexInt, axisInt)
	}
	return true
}

func buildGenericParams(tparams []*ir.Field, num int) []ir.GenericParam {
	genFields := make([]ir.GenericParam, num)
	for i := range num {
		tparamField := tparams[i]
		if ir.IsNonTypeGeneric(tparamField.Type()) {
			genFields[i] = ir.NewGenericNonTypeParam(tparamField)
		} else {
			genFields[i] = ir.NewGenericTypeParam(tparamField)
		}
	}
	return genFields
}

func buildGenericValues(rscope resolveScope, x ir.Expr, ce *compileEvaluator, gen *ir.GenericParams, indices []ir.Expr) ([]ir.GenericValue, bool) {
	gotN, wantN := len(indices), gen.Fields.Len()
	if gotN > wantN {
		return nil, rscope.Err().Appendf(x.Node(), "got %d type arguments but want %d", gotN, wantN)
	}
	genParams := buildGenericParams(gen.Fields.Fields(), len(indices))
	genValues := append([]ir.GenericValue{}, gen.Values...)
	ok := true
	for i, gParam := range genParams {
		numberOk := true
		if !gParam.IsTypeParam() && irkind.IsNumber(indices[i].Type().Kind()) {
			indices[i], numberOk = castNilAndNumber(rscope, indices[i], gParam.Type())
		}
		var assignOk bool
		genValues[gParam.OrigField().Pos], assignOk = gParam.Assign(ce, indices[i])
		ok = ok && assignOk && numberOk
	}
	return genValues, ok
}

func specializeFuncType(rscope resolveScope, x ir.Expr, indices []ir.Expr, fun *ir.FuncValExpr) (ir.Expr, bool) {
	ftype := fun.FuncType()
	if ftype == nil {
		// This is a builtin function with the type built later.
		// That should not be specialised by the user.
		return x, rscope.Err().Appendf(x.Node(), "builtin function does not support type arguments")
	}
	compEval, _, compEvalOk := compEvalForFuncType(rscope, x.Node(), ftype)
	if !compEvalOk {
		return x, false
	}
	if gotN, wantN := len(indices), ftype.GenParams.Fields.Len(); gotN > wantN {
		return x, rscope.Err().Appendf(x.Node(), "got %d type arguments but want %d", gotN, wantN)
	}
	// Specialise the function one type argument at a time:
	// each specialisation is applied on the result of the previous one.
	specX := x
	for _, genExpr := range indices {
		genValues, ok := buildGenericValues(rscope, x, compEval, &fun.FuncType().GenParams, []ir.Expr{genExpr})
		if !ok {
			return x, false
		}
		specTypeExpr := &ir.SpecTypeExpr{
			Src: &ast.IndexExpr{
				X:     specX.Expr(),
				Index: genExpr.Expr(),
			},
			X:     specX,
			Index: genExpr,
		}
		fun, ok = generics.SpecialiseParams(compEval, specTypeExpr, fun, genValues)
		specTypeExpr.Typ = fun.Type()
		specX = fun
		if !ok {
			return specX, false
		}
	}
	return specX, true
}

func specializeFunc(rscope resolveScope, x ir.Expr, indices []ir.Expr) (ir.Expr, bool) {
	var fun *ir.FuncValExpr
	switch xT := x.(type) {
	case *ir.MacroCallExpr:
		fun = ir.NewFuncValExpr(x, xT.F).NewFType(xT.FuncType())
	case ir.WithStore:
		funStorage, ok := xT.Store().(ir.StorageWithValue)
		if !ok {
			return funcError(rscope, x)
		}
		val := funStorage.Value(x)
		valFun, isFun := val.(*ir.FuncValExpr)
		if !isFun {
			return specializeFunc(rscope, val, indices)
		}
		// Keep x (e.g. a method selector) as the callee expression
		// so that the receiver is preserved when the function is evaluated.
		fun = valFun
	case *ir.FuncValExpr:
		fun = xT
	default:
		return invalidExpr(), rscope.Err().AppendInternalf(x.Node(), "cannot specialise function call %T: not supported", x)
	}
	return specializeFuncType(rscope, x, indices, fun)
}

func (n *indexExpr) buildSlicerIndex(rscope resolveScope, ext *ir.IndexExpr, xType ir.SlicerType) bool {
	var ok bool
	ext.Typ, ok = xType.ElementType()
	if !ok {
		return rscope.Err().Appendf(n.source(), "cannot index %s", xType.ReferString(rscope.fileScope().irFile()))
	}
	aType, isArray := xType.(ir.ArrayType)
	boundOk := true
	if isArray {
		boundOk = n.checkIndexBounds(rscope, aType.Rank().Axes()[0], ext.Index)
	}
	var numberOk bool
	ext.Index, numberOk = castNilAndNumber(rscope, ext.Index, ir.Int64Type())
	indexTypeOk := ir.IsIndexType(ext.Index.Type())
	if !indexTypeOk {
		return rscope.Err().Appendf(n.source(), "index %s (of type %s) must be integer", ext.Index.SourceString(rscope.fileScope().irFile()), ext.Index.Type().ReferString(rscope.fileScope().irFile()))
	}
	return boundOk && numberOk && indexTypeOk
}

func (n *indexExpr) buildStructIndex(rscope resolveScope, ext *ir.IndexExpr, xType *ir.StructType) bool {
	indexKind := ext.Index.Type().Kind()
	if indexKind != irkind.Invalid && indexKind != irkind.FieldPath {
		return rscope.Err().Appendf(n.source(), "index %s (of type %s) must be fieldpath", ext.Index.SourceString(rscope.fileScope().irFile()), ext.Index.Type().ReferString(rscope.fileScope().irFile()))
	}
	ext.Typ = ir.UnknownType()
	return true
}

func (n *indexExpr) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	x, xOk := n.x.buildExpr(rscope)
	idx, idxOk := buildCoreExpr(rscope, n.index)
	if !xOk || !idxOk {
		return nil, false
	}
	if irkind.IsNumber(idx.Type().Kind()) {
		idx, idxOk = ir.CastNumber(toFileWithError(rscope), idx, ir.IntType())
	}
	if !idxOk {
		return nil, false
	}
	xType := x.Type()
	if xType.Kind() == irkind.Func {
		return specializeFunc(rscope, x, []ir.Expr{idx})
	}
	ext := &ir.IndexExpr{Src: n.src, X: x, Index: idx, Typ: ir.InvalidType()}
	if xType.Kind() == irkind.Invalid {
		return ext, false
	}
	var ok bool
	switch xTypeT := ir.Underlying(xType).(type) {
	case *ir.SliceType:
		ok = n.buildSlicerIndex(rscope, ext, xTypeT)
	case ir.ArrayType:
		ok = n.buildSlicerIndex(rscope, ext, xTypeT)
	case *ir.StructType:
		ok = n.buildStructIndex(rscope, ext, xTypeT)
	default:
		from := rscope.fileScope().irFile()
		ok = rscope.Err().Appendf(x.Node(), "cannot index %s (type: %s)", ext.X.SourceString(from), xType.ReferString(from))
	}
	return ext, ok
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
	indices := make([]ir.Expr, len(n.indices))
	indicesOk := true
	for i, index := range n.indices {
		var iOk bool
		indices[i], iOk = buildCoreExpr(rscope, index)
		indicesOk = indicesOk && iOk
	}
	if !xOk || !indicesOk {
		return x, false
	}
	if x.Type().Kind() != irkind.Func {
		return x, rscope.Err().Appendf(n.source(), "list of indices only supported for generic functions")
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

// indexExprType is used when a structure is specialised.
type indexExprType struct {
	src *ast.IndexExpr

	x     typeExprNode
	index exprNode
}

var _ typeExprNode = (*indexExprType)(nil)

func processIndexExprType(pscope procScope, src *ast.IndexExpr) (*indexExprType, bool) {
	n := &indexExprType{src: src}
	var xOk, indexOk bool
	n.x, xOk = processTypeExpr(defaultTypeProcScope(pscope), src.X, nil)
	n.index, indexOk = processExpr(pscope, src.Index)
	return n, xOk && indexOk
}

func (n *indexExprType) source() ast.Node {
	return n.src
}

func specializeType(rscope resolveScope, x *ir.TypeValExpr, indices []ir.Expr) (*ir.TypeValExpr, bool) {
	if x.Val().Kind() != irkind.Struct {
		return invalidTypeExprVal, rscope.Err().Appendf(x.Node(), "%s is not a generic type", x.SourceString(rscope.fileScope().irFile()))
	}
	named, err := cast.To[*ir.NamedType](x.Val())
	if err != nil {
		return invalidTypeExprVal, rscope.Err().AppendAt(x.Node(), err)
	}
	ce, ok := rscope.compEval()
	if !ok {
		return invalidTypeExprVal, false
	}
	genValues, ok := buildGenericValues(rscope, x, ce, &named.GenParams, indices)
	if !ok {
		return invalidTypeExprVal, false
	}
	structType, err := cast.To[*ir.StructType](ir.Underlying(named))
	if err != nil {
		return invalidTypeExprVal, rscope.Err().AppendAt(x.Node(), err)
	}
	return generics.SpecialiseStruct(ce, x, structType, genValues)
}

func (n *indexExprType) buildTypeExpr(rscope resolveScope) (*ir.TypeValExpr, bool) {
	xIR, xOk := n.x.buildTypeExpr(rscope)
	idxIR, idxOk := n.index.buildExpr(rscope)
	if !xOk || !idxOk {
		return invalidTypeExprVal, false
	}
	return specializeType(rscope, xIR, []ir.Expr{idxIR})
}

func (n *indexExprType) String() string {
	return fmt.Sprintf("%s[%s]", n.x.String(), n.index.String())

}

// indexExprListType is used when a structure is specialised.
type indexExprListType struct {
	src *ast.IndexListExpr

	x       typeExprNode
	indices []exprNode
}

var _ typeExprNode = (*indexExprListType)(nil)

func processIndexListExprType(pscope procScope, src *ast.IndexListExpr) (*indexExprListType, bool) {
	n := &indexExprListType{src: src}
	var ok bool
	n.x, ok = processTypeExpr(defaultTypeProcScope(pscope), src.X, nil)
	n.indices = make([]exprNode, len(src.Indices))
	for i, idx := range src.Indices {
		var iok bool
		n.indices[i], iok = processExpr(pscope, idx)
		ok = ok && iok
	}
	return n, ok
}

func (n *indexExprListType) source() ast.Node {
	return n.src
}

func (n *indexExprListType) buildTypeExpr(rscope resolveScope) (*ir.TypeValExpr, bool) {
	xIR, ok := n.x.buildTypeExpr(rscope)
	if !ok {
		return invalidTypeExprVal, false
	}
	indices := make([]ir.Expr, len(n.indices))
	for i, idx := range n.indices {
		indices[i], ok = idx.buildExpr(rscope)
		if !ok {
			return invalidTypeExprVal, false
		}
	}
	return specializeType(rscope, xIR, indices)
}

func (n *indexExprListType) String() string {
	ss := make([]string, len(n.indices))
	for i, idx := range n.indices {
		ss[i] = idx.String()
	}
	return fmt.Sprintf("%s[%s]", n.x.String(), strings.Join(ss, ","))
}
