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
	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
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
	indexType := &ir.AtomicType{Knd: toValueT.kind}
	val, unknown, err := ir.Eval[T](fetcher[T]{kind: toValueT.kind, ctx: ctx}, stmt.X)
	if err != nil {
		return nil, true, err
	}
	if len(unknown) > 0 {
		return nil, true, ctx.FileSet().Errorf(stmt.Source(), "unknown variables in static expression")
	}
	ctx.pushBlockFrame()
	defer ctx.popFrame()
	for i := T(0); i < val; i++ {
		iElement, err := state.NewAtomicLiteral(ctx.State(), indexType, i)
		if err != nil {
			return nil, true, err
		}
		if err := ctx.set(stmt.Src.Tok, stmt.Key, iElement); err != nil {
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
	case ir.AxisLengthKind:
		return evalRangeForLoopOverInteger[ir.Int](ctx, stmt, toValue)
	default:
		return nil, true, ctx.FileSet().Errorf(stmt.Source(), "cannot range over %s", xKind.String())
	}
}

func evalRangeStmtForLoopOverArray[T dtype.AlgebraType](ctx *context, stmt *ir.RangeStmt, toValue valuer) (state.Element, bool, error) {
	toValueT := toValue.(valuerT[T])
	indexType := &ir.AtomicType{Knd: toValueT.kind}
	x, err := evalExpr(ctx, stmt.X)
	if err != nil {
		return nil, false, err
	}
	value, ok := x.(state.ArraySlicer)
	if !ok {
		return nil, false, ctx.FileSet().Errorf(stmt.Source(), "cannot range over %T", x)
	}
	arrayShape := value.Shape()
	for i := 0; i < arrayShape.AxisLengths[0]; i++ {
		iElement, err := state.NewAtomicLiteral[T](ctx.State(), indexType, T(i))
		if err != nil {
			return nil, false, err
		}
		iExpr := iElement.ToExpr()
		if err := ctx.set(stmt.Src.Tok, stmt.Key, iElement); err != nil {
			return nil, false, err
		}
		if stmt.Value != nil {
			elementI, err := value.Slice(ctx.exprAt(iExpr), i)
			if err != nil {
				return nil, false, err
			}
			sliceI, _, err := state.NodeFromElement(elementI)
			if err != nil {
				return nil, false, err
			}
			targetShape := &shape.Shape{
				DType:       arrayShape.DType,
				AxisLengths: arrayShape.AxisLengths[1:],
			}
			reshaped, err := ctx.state.BackendGraph().Core().NewReshape(sliceI, targetShape.AxisLengths)
			if err != nil {
				return nil, false, err
			}
			reshapedElement, err := ctx.state.ElementFromNode(ctx.exprAt(iExpr), reshaped, targetShape)
			if err != nil {
				return nil, false, err
			}
			if err := ctx.set(stmt.Src.Tok, stmt.Value, reshapedElement); err != nil {
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
	if kind == ir.TensorKind {
		return evalRangeStmtArray(ctx, stmt)
	}
	return nil, true, errors.Errorf("cannot range over %s", kind.String())
}

func evalIfStmt(ctx *context, stmt *ir.IfStmt) (state.Element, bool, error) {
	ctx.pushBlockFrame()
	defer ctx.popFrame()

	if stmt.Init != nil {
		if _, _, err := evalStmt(ctx, stmt.Init); err != nil {
			return nil, true, err
		}
	}
	cond, err := evalExpr(ctx, stmt.Cond)
	if err != nil {
		return nil, true, err
	}
	condValue, err := state.ConstantScalarFromElement[bool](cond)
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
		if err := ctx.set(stmt.Src.Tok, asg.Dest, cNode); err != nil {
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
		nodes = node.(*state.Tuple).Unpack()
	}
	for i, dest := range stmt.List {
		node := nodes[i]
		if node == nil {
			continue
		}
		ctx.set(stmt.Src.Tok, dest, node)
	}
	return nil
}

func evalReturnStmt(ctx *context, ret *ir.ReturnStmt) (state.Element, bool, error) {
	if len(ret.Results) == 0 {
		// Naked return.
		fields := ctx.currentFrame().function.FuncType().Results.Fields()
		nodes := make([]state.Element, len(fields))
		for i, field := range fields {
			var err error
			nodes[i], err = ctx.find(field.Name)
			if err != nil {
				return nil, false, err
			}
		}
		tuple := toSingleNode(ctx, ret, nodes)
		return tuple, true, nil
	}
	returns := make([]state.Element, len(ret.Results))
	for i, expr := range ret.Results {
		ret, err := evalExpr(ctx, expr)
		if err != nil {
			return nil, false, err
		}
		if ret == nil {
			continue
		}
		returns[i] = ret
	}
	tuple := toSingleNode(ctx, ret, returns)
	return tuple, true, nil
}

func evalValueRef(ctx *context, ref *ir.ValueRef) (state.Element, error) {
	cNode, err := ctx.find(ref.Src)
	if err != nil {
		return nil, err
	}
	return cNode, nil
}

func evalCastToScalarExpr(ctx *context, expr *ir.CastExpr, x state.Element, targetType *ir.AtomicType) (state.Element, error) {
	constant := state.ConstantFromElement(x)
	if constant != nil {
		return evalScalarCastOnHost(ctx, expr, constant, targetType.Kind())
	}
	xNode, _, err := state.NodeFromElement(x)
	if err != nil {
		return nil, err
	}
	targetShape := &shape.Shape{
		DType: targetType.Kind().DType(),
	}
	reshaped, err := ctx.state.BackendGraph().Core().NewReshape(xNode, nil)
	if err != nil {
		return nil, err
	}
	castNode, err := ctx.state.BackendGraph().Core().NewCast(reshaped, targetShape.DType)
	if err != nil {
		return nil, err
	}
	return ctx.state.ElementFromNode(ctx.exprAt(expr), castNode, targetShape)
}

func evalCastToArrayExpr(ctx *context, expr *ir.CastExpr, x state.Element, targetType *ir.ArrayType) (state.Element, error) {
	xNode, xShape, err := state.NodeFromElement(x)
	if err != nil {
		return nil, err
	}
	bckGraph := ctx.state.BackendGraph()
	targetDType := targetType.DataType().Kind().DType()
	if genericRank, isGeneric := targetType.Rank().(*ir.GenericRank); isGeneric && genericRank.IsGeneric() {
		targetShape := &shape.Shape{
			DType:       targetDType,
			AxisLengths: xShape.AxisLengths,
		}
		castNode, err := ctx.state.BackendGraph().Core().NewCast(xNode, targetShape.DType)
		if err != nil {
			return nil, err
		}
		return ctx.state.ElementFromNode(ctx.exprAt(expr), castNode, targetShape)
	}
	if xShape.DType != targetDType {
		xNode, err = bckGraph.Core().NewCast(xNode, targetDType)
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
		size, unknowns, err := ir.Eval[ir.Int](fetcher, dim.Expr())
		if err != nil {
			return nil, err
		}
		if len(unknowns) > 0 {
			return nil, ctx.FileSet().Errorf(expr.Src, "cannot reshape %s: %v unknown", targetType, unknowns)
		}
		dimensions[i] = int(size)
	}
	reshape, err := bckGraph.Core().NewReshape(xNode, dimensions)
	if err != nil {
		return nil, ctx.FileSet().Position(expr.Src, err)
	}
	sourceType, ok := expr.X.Type().(*ir.ArrayType)
	if !ok {
		return nil, ctx.FileSet().Errorf(expr.Src, "cannot cast %T to *ir.ArrayType", sourceType)
	}
	var resultNode graph.Node = reshape
	if sourceType.DType.Kind().DType() != targetDType {
		resultNode, err = bckGraph.Core().NewCast(reshape, targetDType)
		if err != nil {
			return nil, err
		}
	}
	return ctx.state.ElementFromNode(ctx.exprAt(expr), resultNode, &shape.Shape{
		DType:       targetDType,
		AxisLengths: dimensions,
	})
}

func evalCastExpr(ctx *context, expr *ir.CastExpr) (state.Element, error) {
	x, err := evalExpr(ctx, expr.X)
	if err != nil {
		return nil, err
	}
	switch targetType := expr.Typ.(type) {
	case *ir.ArrayType:
		return evalCastToArrayExpr(ctx, expr, x, targetType)
	case *ir.AtomicType:
		return evalCastToScalarExpr(ctx, expr, x, targetType)
	default:
		return nil, ctx.FileSet().Errorf(expr.Src, "cast to %T not supported", targetType)
	}
}

func evalUnaryExpression(ctx *context, expr *ir.UnaryExpr) (state.Element, error) {
	x, err := evalExpr(ctx, expr.X)
	if err != nil {
		return nil, err
	}
	if ir.IsStatic(expr.Type()) {
		return nil, errors.Errorf("not supported")
	}
	xNode, xShape, err := state.NodeFromElement(x)
	if err != nil {
		return nil, err
	}
	unaryNode, err := ctx.state.BackendGraph().Core().NewUnary(expr.Src, xNode)
	if err != nil {
		return nil, err
	}
	return ctx.state.ElementFromNode(ctx.exprAt(expr), unaryNode, xShape)
}

func atomicExprFromElement(ctx *context, el state.Element) ir.Expr {
	array, ok := el.(state.ElementWithConstant)
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
	xNode, xShape, err := state.NodeFromElement(x)
	if err != nil {
		return nil, err
	}
	yNode, yShape, err := state.NodeFromElement(y)
	if err != nil {
		return nil, err
	}
	binaryNode, err := ctx.state.BackendGraph().Core().NewBinary(expr.Src, xNode, yNode)
	if err != nil {
		return nil, err
	}
	targetShape := &shape.Shape{
		DType:       xShape.DType,
		AxisLengths: xShape.AxisLengths,
	}
	if ir.IsBoolOp(expr.Src.Op) {
		targetShape.DType = dtype.Bool
	}
	if len(yShape.AxisLengths) > 0 {
		targetShape.AxisLengths = yShape.AxisLengths
	}
	return ctx.state.ElementFromNode(ctx.exprAt(expr), binaryNode, targetShape)
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
	return ctx.state.Methods(node)
}

func evalStructLiteral(ctx *context, expr *ir.StructLitExpr) (state.Element, error) {
	under := ir.Underlying(expr.Typ)
	structType, ok := under.(*ir.StructType)
	if !ok {
		return nil, ctx.FileSet().Errorf(expr.Source(), "underlying type %T is not a structure", expr.Typ)
	}
	fields := make([]state.Element, structType.NumFields())
	for _, fieldLit := range expr.Elts {
		node, err := evalExpr(ctx, fieldLit.X)
		if err != nil {
			return nil, err
		}
		fields[fieldLit.Field.ID] = node
	}
	var node state.Element = ctx.state.Struct(structType, ctx.exprAt(expr), fields)
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
	elements := make([]state.Element, len(expr.Vals))
	for i, expr := range expr.Vals {
		elt, err := evalExpr(ctx, expr)
		if err != nil {
			return nil, err
		}
		elements[i] = elt
	}
	return ctx.state.ToSlice(ctx.exprAt(expr), elements), nil
}

func evalExpr(ctx *context, expr ir.Expr) (state.Element, error) {
	if expr == nil {
		return nil, errors.Errorf("cannot evaluate a nil expression")
	}
	switch exprT := expr.(type) {
	case ir.Atomic:
		return evalScalarLiteral(ctx, exprT)
	case ir.ArrayLitExpr:
		return evalArrayLiteral(ctx, exprT)
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
	default:
		return nil, ctx.FileSet().Errorf(expr.Source(), "cannot evaluate GX expression: %T not supported", expr)
	}
}

func evalFieldSelectorExpr(ctx *context, ref *ir.FieldSelectorExpr) (state.Element, error) {
	node, err := evalExpr(ctx, ref.X)
	if err != nil {
		return nil, err
	}
	slt, ok := node.(state.FieldSelector)
	if !ok {
		return nil, ctx.FileSet().Errorf(ref.Source(), "%T has no field %s", node, ref.Src.Sel.Name)
	}
	return slt.SelectField(ctx.exprAt(ref), ref.Field.ID)
}

func evalFuncLit(ctx *context, ref *ir.FuncLit) (state.Element, error) {
	return ctx.state.Func(ref, nil), nil
}

func evalFuncLitCall(ctx *context, ref *ir.FuncLit, args []state.Element) (state.Element, error) {
	graph := ctx.State().BackendGraph()
	subgraph, err := graph.Core().NewSubgraph(ref.Name())
	if err != nil {
		return nil, err
	}
	subctx := newContext(ctx.itrp, state.New(ref, subgraph), ref, nil, nil)
	funcFrame, err := subctx.pushFuncFrame(ref)
	if err != nil {
		return nil, err
	}
	defer subctx.popFrame()
	assignArgumentValues(ref.FuncType(), funcFrame, args)
	for _, resultName := range fieldNames(ref.FType.Results.List) {
		funcFrame.define(resultName.Name, nil)
	}

	subresults, err := evalFuncBody(subctx, ref.Body)
	if err != nil {
		return nil, err
	}
	resultNode, shape, err := state.NodeFromElement(subresults)
	if err != nil {
		return nil, err
	}
	result, err := graph.Core().NewCall(subgraph, resultNode)
	if err != nil {
		return nil, err
	}
	return ctx.State().ElementFromNode(ctx.exprAt(ref), result, shape)
}

func evalIndexExpr(ctx *context, ref *ir.IndexExpr) (state.Element, error) {
	x, err := evalExpr(ctx, ref.X)
	if err != nil {
		return nil, err
	}
	slicer, ok := x.(state.Slicer)
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
	return slicer.Slice(ctx.exprAt(ref), index)
}

func computeEinsumAxisLengths(ref *ir.EinsumExpr, xShape, yShape *shape.Shape, node graph.Node) []int {
	// TODO(degris): hack to postpone computing einstein sum in the interpreter.
	// We rely on PJRT for the moment.
	return node.(interface{ PJRTDims() []int }).PJRTDims()
}

func evalEinsumExpr(ctx *context, ref *ir.EinsumExpr) (state.Element, error) {
	x, xShape, err := ctx.evalNumericalExpr(ref.X)
	if err != nil {
		return nil, err
	}
	y, yShape, err := ctx.evalNumericalExpr(ref.Y)
	if err != nil {
		return nil, err
	}
	dotNode, err := ctx.State().BackendGraph().Core().NewDotGeneral(x, y, ref.BatchAxes, ref.ReduceAxes)
	if err != nil {
		return nil, err
	}
	targetShape := &shape.Shape{
		DType:       xShape.DType,
		AxisLengths: computeEinsumAxisLengths(ref, xShape, yShape, dotNode),
	}
	return ctx.State().ElementFromNode(ctx.exprAt(ref), dotNode, targetShape)
}

func evalMethodSelectorExpr(ctx *context, ref *ir.MethodSelectorExpr) (state.Element, error) {
	node, err := evalExpr(ctx, ref.X)
	if err != nil {
		return nil, err
	}
	slt, ok := node.(state.MethodSelector)
	if !ok {
		return nil, ctx.FileSet().Errorf(ref.Source(), "%T(%v) has no method %s", node, node, ref.Src.Sel.Name)
	}
	return slt.SelectMethod(ref.Func)
}

func evalPackageFuncSelectorExpr(ctx *context, ref *ir.PackageFuncSelectorExpr) (state.Element, error) {
	pkgIdent := ref.Package.Decl.Name()
	node, err := ctx.find(pkgIdent)
	if err != nil {
		return nil, err
	}
	pkg, ok := node.(*state.Package)
	if !ok {
		return nil, ctx.FileSet().Errorf(ref.Source(), "undefined: %s.%s", pkgIdent.Name, ref.Src.Sel.Name)
	}
	return pkg.SelectMethod(ref.Func)
}

func evalPackageTypeSelector(ctx *context, ref *ir.PackageTypeSelector) (state.Element, error) {
	pkgIdent := ref.Package.Decl.Name()
	node, err := ctx.find(pkgIdent)
	if err != nil {
		return nil, err
	}
	pkg, ok := node.(*state.Package)
	if !ok {
		return nil, ctx.FileSet().Errorf(ref.Source(), "undefined: %s.%s", pkgIdent.Name, ref.Src.Sel.Name)
	}
	return pkg.SelectType(ref.Typ), nil
}

func evalPackageConstSelectorExpr(ctx *context, ref *ir.PackageConstSelectorExpr) (state.Element, error) {
	return evalExpr(ctx, ref.Const.Value)
}
