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

package interp

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/state"
)

func evalBlockStmt(ctx *context, body *ir.BlockStmt) (state.Element, bool, error) {
	var output state.Element
	var stop bool
	for _, node := range body.List {
		var err error
		output, stop, err = evalStmt(ctx, node)
		if err != nil {
			return nil, true, err
		}
		if stop {
			break
		}
	}
	return output, stop, nil
}

func evalStmt(ctx *context, node ir.Stmt) (state.Element, bool, error) {
	switch nodeT := node.(type) {
	case *ir.AssignCallStmt:
		return nil, false, evalAssignCallStmt(ctx, nodeT)
	case *ir.AssignExprStmt:
		return nil, false, evalAssignExprStmt(ctx, nodeT)
	case *ir.RangeStmt:
		return evalRangeStmt(ctx, nodeT)
	case *ir.IfStmt:
		return evalIfStmt(ctx, nodeT)
	case *ir.ReturnStmt:
		return evalReturnStmt(ctx, nodeT)
	case *ir.BlockStmt:
		return evalBlockStmt(ctx, nodeT)
	case *ir.ExprStmt:
		_, err := evalExpr(ctx, nodeT.X)
		return nil, false, err
	default:
		return nil, false, ctx.FileSet().Errorf(node.Source(), "cannot evaluate GX node: %T not supported", node)
	}
}

func evalRangeForLoopOverInteger[T dtype.AlgebraType](ctx *context, stmt *ir.RangeStmt, toValue valuer) (state.Element, bool, error) {
	toValueT := toValue.(valuerT[T])
	indexType := ir.TypeFromKind(toValueT.kind)
	val, unknown, err := ir.Eval[T](fetcher[T]{kind: toValueT.kind, ctx: ctx}, stmt.X)
	if err != nil {
		return nil, true, err
	}
	if len(unknown) > 0 {
		return nil, true, ctx.FileSet().Errorf(stmt.Source(), "unknown variables in static expression")
	}
	ctx.frame().pushBlockFrame()
	defer ctx.frame().popFrame()
	for i := T(0); i < val; i++ {
		iExpr := &ir.AtomicValueT[T]{
			Src: stmt.Key.Expr(),
			Val: i,
			Typ: indexType,
		}
		iValue := toValueT.toAtomValue(iExpr.Type(), i)
		iElement, err := ctx.Evaluator().ElementFromValue(nodeAt[ir.Node](ctx, iExpr), iValue)
		if err != nil {
			return nil, true, err
		}
		if err := ctx.frame().set(stmt.Src.Tok, stmt.Key, iElement); err != nil {
			return nil, false, err
		}
		element, stop, err := evalBlockStmt(ctx, stmt.Body)
		if stop || err != nil {
			return element, stop, err
		}
	}
	return nil, false, nil
}

func evalRangeStmtInteger(ctx *context, stmt *ir.RangeStmt, xKind ir.Kind) (state.Element, bool, error) {
	toValue, err := newValuer(ctx, stmt.X, xKind)
	if err != nil {
		return nil, false, err
	}
	switch xKind {
	case ir.IntLenKind:
		return evalRangeForLoopOverInteger[ir.Int](ctx, stmt, toValue)
	default:
		return nil, true, ctx.FileSet().Errorf(stmt.Source(), "cannot range over %s", xKind.String())
	}
}

func evalRangeStmtForLoopOverArray[T dtype.AlgebraType](ctx *context, stmt *ir.RangeStmt, toValue valuer) (state.Element, bool, error) {
	toValueT := toValue.(valuerT[T])
	indexType := ir.TypeFromKind(toValueT.kind)
	x, err := evalExpr(ctx, stmt.X)
	if err != nil {
		return nil, false, err
	}
	value, ok := x.(elements.ArraySlicer)
	if !ok {
		return nil, false, ctx.FileSet().Errorf(stmt.Source(), "cannot range over %T", x)
	}
	arrayShape := value.Shape()
	for i := 0; i < arrayShape.AxisLengths[0]; i++ {
		iExpr := &ir.AtomicValueT[T]{
			Src: stmt.Key.Expr(),
			Val: T(i),
			Typ: indexType,
		}
		iValue := toValueT.toAtomValue(iExpr.Type(), T(i))
		iElement, err := ctx.Evaluator().ElementFromValue(nodeAt[ir.Node](ctx, iExpr), iValue)
		if err != nil {
			return nil, false, err
		}
		if err := ctx.frame().set(stmt.Src.Tok, stmt.Key, iElement); err != nil {
			return nil, false, err
		}
		if stmt.Value != nil {
			elementI, err := value.Slice(ctx.ExprAt(iExpr), i)
			if err != nil {
				return nil, false, err
			}
			reshapedElement, err := ctx.Evaluator().Reshape(ctx, iExpr, elementI, arrayShape.AxisLengths[1:])
			if err != nil {
				return nil, false, err
			}
			if err := ctx.frame().set(stmt.Src.Tok, stmt.Value, reshapedElement); err != nil {
				return nil, false, err
			}
		}
		element, stop, err := evalBlockStmt(ctx, stmt.Body)
		if stop || err != nil {
			return element, stop, err
		}
	}
	return nil, false, nil
}

func evalRangeStmtArray(ctx *context, stmt *ir.RangeStmt) (state.Element, bool, error) {
	keyKind := stmt.Key.Type().Kind()
	toValue, err := newValuer(ctx, stmt.X, keyKind)
	if err != nil {
		return nil, false, err
	}
	switch keyKind {
	case ir.Int64Kind:
		return evalRangeStmtForLoopOverArray[ir.Int](ctx, stmt, toValue)
	default:
		return nil, true, ctx.FileSet().Errorf(stmt.Source(), "cannot range over %s", keyKind.String())
	}
}

func evalRangeStmt(ctx *context, stmt *ir.RangeStmt) (state.Element, bool, error) {
	kind := stmt.X.Type().Kind()
	if ir.IsRangeOk(kind) {
		return evalRangeStmtInteger(ctx, stmt, kind)
	}
	if kind == ir.ArrayKind {
		return evalRangeStmtArray(ctx, stmt)
	}
	return nil, true, errors.Errorf("cannot range over %s", kind.String())
}

func evalIfStmt(ctx *context, stmt *ir.IfStmt) (state.Element, bool, error) {
	ctx.frame().pushBlockFrame()
	defer ctx.frame().popFrame()

	if stmt.Init != nil {
		if _, _, err := evalStmt(ctx, stmt.Init); err != nil {
			return nil, true, err
		}
	}
	cond, err := evalExpr(ctx, stmt.Cond)
	if err != nil {
		return nil, true, err
	}
	condValue, err := elements.ConstantScalarFromElement[bool](cond)
	if err != nil {
		return nil, true, err
	}
	if condValue {
		return evalBlockStmt(ctx, stmt.Body)
	}
	if stmt.Else == nil {
		return nil, false, nil
	}
	return evalStmt(ctx, stmt.Else)
}

func evalAssignExprStmt(ctx *context, stmt *ir.AssignExprStmt) error {
	for _, asg := range stmt.List {
		cNode, err := evalExpr(ctx, asg.Expr)
		if err != nil {
			return err
		}
		if cNode == nil {
			continue
		}
		if err := ctx.frame().set(stmt.Src.Tok, asg.Dest, cNode); err != nil {
			return err
		}
	}
	return nil
}

func evalAssignCallStmt(ctx *context, stmt *ir.AssignCallStmt) error {
	node, err := evalCallExpr(ctx, stmt.Call)
	if err != nil {
		return err
	}
	var nodes []state.Element
	if len(stmt.List) == 1 {
		nodes = []state.Element{node}
	} else {
		nodes = node.(*elements.Tuple).Elements()
	}
	for i, dest := range stmt.List {
		node := nodes[i]
		if node == nil {
			continue
		}
		ctx.frame().set(stmt.Src.Tok, dest, node)
	}
	return nil
}

func evalReturnStmt(ctx *context, ret *ir.ReturnStmt) (state.Element, bool, error) {
	if len(ret.Results) == 0 {
		// Naked return.
		fields := ctx.frame().currentFrame().Context().function.FuncType().Results.Fields()
		nodes := make([]state.Element, len(fields))
		for i, field := range fields {
			var err error
			nodes[i], err = ctx.frame().find(field.Name)
			if err != nil {
				return nil, false, err
			}
		}
		tuple := toSingleNode(ctx, ret, nodes)
		return tuple, true, nil
	}
	returns := make([]state.Element, len(ret.Results))
	for i, expr := range ret.Results {
		var err error
		returns[i], err = evalExpr(ctx, expr)
		if err != nil {
			return nil, false, err
		}
	}
	tuple := toSingleNode(ctx, ret, returns)
	return tuple, true, nil
}

func evalValueRef(ctx *context, ref *ir.ValueRef) (state.Element, error) {
	cNode, err := ctx.frame().find(ref.Src)
	if err != nil {
		return nil, err
	}
	return cNode, nil
}

func evalCastToScalarExpr(ctx *context, expr *ir.CastExpr, x state.Element, targetType ir.ArrayType) (state.Element, error) {
	constant := elements.ConstantFromElement(x)
	if constant != nil {
		return evalScalarCastOnHost(ctx, expr, constant, targetType.Kind())
	}
	targetShape := &shape.Shape{
		DType: targetType.Kind().DType(),
	}
	reshaped, err := ctx.Evaluator().Reshape(ctx, expr, x, nil)
	if err != nil {
		return nil, err
	}
	return ctx.Evaluator().Cast(ctx, expr, reshaped, targetShape.DType)
}

func evalCastToArrayExpr(ctx *context, expr *ir.CastExpr, x state.Element, targetType ir.ArrayType) (state.Element, error) {
	_, xDType := ir.Shape(expr.X.Type())
	targetDType := targetType.DataType().Kind().DType()
	if genericRank, isGeneric := targetType.Rank().(*ir.GenericRank); isGeneric && genericRank.IsGeneric() {
		return ctx.Evaluator().Cast(ctx, expr, x, targetDType)
	}
	if xDType.Kind().DType() != targetDType {
		var err error
		x, err = ctx.Evaluator().Cast(ctx, expr, x, targetDType)
		if err != nil {
			return nil, err
		}
	}
	rank, err := rankOf(ctx, expr, targetType)
	if err != nil {
		return nil, err
	}
	dims := rank.Axes
	fetcher := &fetcher[ir.Int]{kind: ir.DefaultIntKind, ctx: ctx}
	dimensions := make([]int, len(dims))
	for i, dim := range dims {
		dimExpr := dim.Expr()
		if dimExpr == nil {
			return nil, ctx.FileSet().Errorf(expr.Src, "cannot reshape an array with an unknown axis length")
		}
		size, unknowns, err := ir.Eval[ir.Int](fetcher, dim)
		if err != nil {
			return nil, err
		}
		if len(unknowns) > 0 {
			return nil, ctx.FileSet().Errorf(expr.Src, "cannot reshape %s: %v unknown", targetType, unknowns)
		}
		dimensions[i] = int(size)
	}
	reshape, err := ctx.Evaluator().Reshape(ctx, expr, x, dimensions)
	if err != nil {
		return nil, ctx.FileSet().Position(expr.Src, err)
	}
	sourceType, ok := expr.X.Type().(ir.ArrayType)
	if !ok {
		return nil, ctx.FileSet().Errorf(expr.Src, "cannot cast %T to %s", sourceType, reflect.TypeFor[ir.ArrayType]().Name())
	}
	if sourceType.DataType().Kind().DType() == targetDType {
		return reshape, nil
	}
	return ctx.Evaluator().Cast(ctx, expr, reshape, targetDType)
}

func evalCastExpr(ctx *context, expr *ir.CastExpr) (state.Element, error) {
	x, err := evalExpr(ctx, expr.X)
	if err != nil {
		return nil, err
	}
	arrayType, ok := expr.Typ.(ir.ArrayType)
	if !ok {
		return nil, ctx.FileSet().Errorf(expr.Src, "cast to %s not supported", reflect.TypeFor[ir.ArrayType]().Name())
	}
	if arrayType.Rank().IsAtomic() {
		return evalCastToScalarExpr(ctx, expr, x, arrayType)
	}
	return evalCastToArrayExpr(ctx, expr, x, arrayType)
}

func evalUnaryExpression(ctx *context, expr *ir.UnaryExpr) (state.Element, error) {
	x, err := evalExpr(ctx, expr.X)
	if err != nil {
		return nil, err
	}
	if ir.IsStatic(expr.Type()) {
		return nil, errors.Errorf("not supported")
	}
	return ctx.Evaluator().UnaryOp(ctx, expr, x)
}

func atomicExprFromElement(ctx *context, el state.Element) ir.Expr {
	array, ok := el.(elements.ElementWithConstant)
	if !ok {
		return nil
	}
	if !array.Shape().IsAtomic() {
		return nil
	}
	return array.ToExpr()
}

func evalBinaryExpression(ctx *context, expr *ir.BinaryExpr) (state.Element, error) {
	x, err := evalExpr(ctx, expr.X)
	if err != nil {
		return nil, err
	}
	y, err := evalExpr(ctx, expr.Y)
	if err != nil {
		return nil, err
	}
	xHost := atomicExprFromElement(ctx, x)
	yHost := atomicExprFromElement(ctx, y)
	if xHost != nil && yHost != nil {
		return evalBinaryStaticExpression(ctx, expr, xHost, yHost)
	}
	return ctx.Evaluator().BinaryOp(ctx, expr, x, y)
}

func evalBinaryStaticExpression(ctx *context, expr *ir.BinaryExpr, x, y ir.Expr) (state.Element, error) {
	exprForEval := &ir.BinaryExpr{
		Src: expr.Src,
		X:   x,
		Y:   y,
		Typ: expr.Typ,
	}
	valuer, err := newValuer(ctx, exprForEval, exprForEval.Type().Kind())
	if err != nil {
		return nil, err
	}
	return valuer.eval(ctx, exprForEval)
}

func evalNamedType(ctx *context, tp *ir.NamedType, node state.Element) state.Element {
	if len(tp.Methods) == 0 {
		return node
	}
	return elements.NewMethods(node)
}

func evalStructLiteral(ctx *context, expr *ir.StructLitExpr) (state.Element, error) {
	under := ir.Underlying(expr.Typ)
	structType, ok := under.(*ir.StructType)
	if !ok {
		return nil, ctx.FileSet().Errorf(expr.Source(), "underlying type %T is not a structure", expr.Typ)
	}
	fields := make(map[string]state.Element, structType.NumFields())
	for _, fieldLit := range expr.Elts {
		node, err := evalExpr(ctx, fieldLit.X)
		if err != nil {
			return nil, err
		}
		fields[fieldLit.Field.Name.Name] = node
	}
	var node state.Element = elements.NewStruct(structType, ctx.ExprAt(expr), fields)
	var err error
	switch tp := expr.Typ.(type) {
	case *ir.NamedType:
		node = evalNamedType(ctx, tp, node)
	default:
		err = ctx.FileSet().Errorf(expr.Source(), "invalid structure literal type: %T", expr.Typ)
	}
	if err != nil {
		return nil, err
	}
	return node, nil
}

func evalSliceLiteral(ctx *context, expr *ir.SliceExpr) (state.Element, error) {
	els := make([]state.Element, len(expr.Vals))
	for i, expr := range expr.Vals {
		elt, err := evalExpr(ctx, expr)
		if err != nil {
			return nil, err
		}
		els[i] = elt
	}
	return elements.ToSlice(ctx.ExprAt(expr), els), nil
}

func evalExpr(ctx *context, expr ir.Expr) (state.Element, error) {
	if expr == nil {
		return nil, errors.Errorf("cannot evaluate a nil expression")
	}
	switch exprT := expr.(type) {
	case *ir.ArrayLitExpr:
		return evalArrayLiteral(ctx, exprT)
	case *ir.NumberCastExpr:
		return evalScalarLiteral(ctx, exprT)
	case *ir.SliceExpr:
		return evalSliceLiteral(ctx, exprT)
	case *ir.StructLitExpr:
		return evalStructLiteral(ctx, exprT)
	case *ir.CastExpr:
		return evalCastExpr(ctx, exprT)
	case *ir.CallExpr:
		return evalCallExpr(ctx, exprT)
	case *ir.UnaryExpr:
		return evalUnaryExpression(ctx, exprT)
	case *ir.ParenExpr:
		return evalExpr(ctx, exprT.X)
	case *ir.BinaryExpr:
		return evalBinaryExpression(ctx, exprT)
	case *ir.ValueRef:
		return evalValueRef(ctx, exprT)
	case *ir.FieldSelectorExpr:
		return evalFieldSelectorExpr(ctx, exprT)
	case *ir.FuncLit:
		return evalFuncLit(ctx, exprT)
	case *ir.IndexExpr:
		return evalIndexExpr(ctx, exprT)
	case *ir.EinsumExpr:
		return evalEinsumExpr(ctx, exprT)
	case *ir.MethodSelectorExpr:
		return evalMethodSelectorExpr(ctx, exprT)
	case *ir.PackageFuncSelectorExpr:
		return evalPackageFuncSelectorExpr(ctx, exprT)
	case *ir.PackageTypeSelector:
		return evalPackageTypeSelector(ctx, exprT)
	case *ir.PackageConstSelectorExpr:
		return evalPackageConstSelectorExpr(ctx, exprT)
	case ir.StaticExpr:
		return evalScalarLiteral(ctx, exprT)
	default:
		return nil, ctx.FileSet().Errorf(expr.Source(), "cannot evaluate GX expression: %T not supported", expr)
	}
}

func evalFieldSelectorExpr(ctx *context, ref *ir.FieldSelectorExpr) (state.Element, error) {
	node, err := evalExpr(ctx, ref.X)
	if err != nil {
		return nil, err
	}
	slt, ok := node.(elements.FieldSelector)
	if !ok {
		return nil, ctx.FileSet().Errorf(ref.Source(), "%T has no field %s", node, ref.Src.Sel.Name)
	}
	return slt.SelectField(ctx.ExprAt(ref), ref.Field.Name.Name)
}

func evalFuncLit(ctx *context, ref *ir.FuncLit) (state.Element, error) {
	return elements.NewFunc(ref, nil), nil
}

func evalIndexExpr(ctx *context, ref *ir.IndexExpr) (state.Element, error) {
	x, err := evalExpr(ctx, ref.X)
	if err != nil {
		return nil, err
	}
	slicer, ok := x.(elements.Slicer)
	if !ok {
		return nil, ctx.FileSet().Errorf(ref.Source(), "cannot index over %T", x)
	}
	indexValue, err := evalExprOnHost(ctx, ref.Index)
	if err != nil {
		return nil, err
	}
	index, err := toGoInt(indexValue)
	if err != nil {
		return nil, ctx.FileSet().Errorf(ref.Source(), "cannot cast value %T to an index: %v", indexValue, err)
	}
	return slicer.Slice(ctx.ExprAt(ref), index)
}

func evalEinsumExpr(ctx *context, ref *ir.EinsumExpr) (state.Element, error) {
	x, err := evalExpr(ctx, ref.X)
	if err != nil {
		return nil, err
	}
	y, err := evalExpr(ctx, ref.Y)
	if err != nil {
		return nil, err
	}
	return ctx.Evaluator().Einsum(ctx, ref, x, y)
}

func evalMethodSelectorExpr(ctx *context, ref *ir.MethodSelectorExpr) (state.Element, error) {
	node, err := evalExpr(ctx, ref.X)
	if err != nil {
		return nil, err
	}
	slt, ok := node.(elements.MethodSelector)
	if !ok {
		return nil, ctx.FileSet().Errorf(ref.Source(), "%T(%v) has no method %s", node, node, ref.Src.Sel.Name)
	}
	return slt.SelectMethod(ref.Func)
}

func evalPackageFuncSelectorExpr(ctx *context, ref *ir.PackageFuncSelectorExpr) (state.Element, error) {
	pkgIdent := ref.Package.Decl.Name()
	node, err := ctx.frame().find(pkgIdent)
	if err != nil {
		return nil, err
	}
	pkg, ok := node.(*elements.Package)
	if !ok {
		return nil, ctx.FileSet().Errorf(ref.Source(), "undefined: %s.%s", pkgIdent.Name, ref.Src.Sel.Name)
	}
	return pkg.SelectMethod(ref.Func)
}

func evalPackageTypeSelector(ctx *context, ref *ir.PackageTypeSelector) (state.Element, error) {
	pkgIdent := ref.Package.Decl.Name()
	node, err := ctx.frame().find(pkgIdent)
	if err != nil {
		return nil, err
	}
	pkg, ok := node.(*elements.Package)
	if !ok {
		return nil, ctx.FileSet().Errorf(ref.Source(), "undefined: %s.%s", pkgIdent.Name, ref.Src.Sel.Name)
	}
	return pkg.SelectType(ref.Typ), nil
}

func evalPackageConstSelectorExpr(ctx *context, ref *ir.PackageConstSelectorExpr) (state.Element, error) {
	return evalExpr(ctx, ref.Const.Value)
}
