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
	"go/ast"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/numbers"
)

func evalBlockStmt(ctx *context, body *ir.BlockStmt) ([]elements.Element, bool, error) {
	var outs []elements.Element
	var stop bool
	for _, node := range body.List {
		var err error
		outs, stop, err = evalStmt(ctx, node)
		if err != nil {
			return nil, true, err
		}
		if stop {
			break
		}
	}
	return outs, stop, nil
}

func evalStmt(ctx *context, node ir.Stmt) ([]elements.Element, bool, error) {
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
		_, err := ctx.evalExpr(nodeT.X)
		return nil, false, err
	default:
		return nil, false, fmterr.Errorf(ctx.File().FileSet(), node.Source(), "cannot evaluate GX node: %T not supported", node)
	}
}

func evalRangeForLoopOverInteger[T dtype.AlgebraType](ctx *context, stmt *ir.RangeStmt, toValue valuer) ([]elements.Element, bool, error) {
	toValueT := toValue.(valuerT[T])
	indexType := ir.TypeFromKind(toValueT.kind)
	val, err := evalAtom[T](ctx, stmt.X)
	if err != nil {
		return nil, true, err
	}
	ctx.pushBlockFrame()
	defer ctx.popFrame()
	for i := T(0); i < val; i++ {
		iExpr := &ir.AtomicValueT[T]{
			Src: stmt.Key.Source().(ast.Expr),
			Val: i,
			Typ: indexType,
		}
		iValue, err := toValueT.toAtomValue(iExpr.Type(), i)
		if err != nil {
			return nil, true, err
		}
		iElement, err := ctx.eval.evaluator.ElementFromAtom(elements.NewExprAt(ctx.File(), iExpr), iValue)
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

func evalRangeStmtInteger(ctx *context, stmt *ir.RangeStmt, xKind ir.Kind) ([]elements.Element, bool, error) {
	toValue, err := newValuer(ctx, stmt.X, xKind)
	if err != nil {
		return nil, false, err
	}
	switch xKind {
	case ir.IntLenKind:
		return evalRangeForLoopOverInteger[ir.Int](ctx, stmt, toValue)
	default:
		return nil, true, fmterr.Errorf(ctx.File().FileSet(), stmt.Source(), "cannot range over %s", xKind.String())
	}
}

func evalRangeStmtForLoopOverArray[T dtype.AlgebraType](ctx *context, stmt *ir.RangeStmt, toValue valuer) ([]elements.Element, bool, error) {
	toValueT := toValue.(valuerT[T])
	indexType := ir.TypeFromKind(toValueT.kind)
	x, err := ctx.evalExpr(stmt.X)
	if err != nil {
		return nil, false, err
	}
	value, ok := x.(elements.ArraySlicer)
	if !ok {
		return nil, false, fmterr.Errorf(ctx.File().FileSet(), stmt.Source(), "cannot range over %T", x)
	}
	arrayShape := value.Shape()
	for i := 0; i < arrayShape.AxisLengths[0]; i++ {
		iExpr := &ir.AtomicValueT[T]{
			Src: stmt.Key.Source().(ast.Expr),
			Val: T(i),
			Typ: indexType,
		}
		iValue, err := toValueT.toAtomValue(iExpr.Type(), T(i))
		if err != nil {
			return nil, false, err
		}
		iElement, err := ctx.eval.evaluator.ElementFromAtom(elements.NewExprAt(ctx.File(), iExpr), iValue)
		if err != nil {
			return nil, false, err
		}
		if err := ctx.set(stmt.Src.Tok, stmt.Key, iElement); err != nil {
			return nil, false, err
		}
		if stmt.Value != nil {
			valueExpr := &ir.SliceLitExpr{
				Src: stmt.Src.X,
				Typ: value.Type(), // TODO(396633820): compute the correct type (the dtype will be correct but not the shape)
			}
			elementI, err := value.SliceArray(ctx, valueExpr, iElement)
			if err != nil {
				return nil, false, err
			}
			dims, err := dimsAsElements(ctx, valueExpr, arrayShape.AxisLengths[1:])
			if err != nil {
				return nil, false, err
			}
			reshapedElement, err := ctx.eval.evaluator.ArrayOps().Reshape(elements.NewExprAt(ctx.File(), valueExpr), elementI, dims)
			if err != nil {
				return nil, false, err
			}
			if err := ctx.set(stmt.Src.Tok, stmt.Value, reshapedElement); err != nil {
				return nil, false, err
			}
		}
		outs, stop, err := evalBlockStmt(ctx, stmt.Body)
		if stop || err != nil {
			return outs, stop, err
		}
	}
	return nil, false, nil
}

func evalRangeStmtArray(ctx *context, stmt *ir.RangeStmt) ([]elements.Element, bool, error) {
	keyKind := stmt.Key.Type().Kind()
	toValue, err := newValuer(ctx, stmt.X, keyKind)
	if err != nil {
		return nil, false, err
	}
	switch keyKind {
	case ir.Int64Kind:
		return evalRangeStmtForLoopOverArray[ir.Int](ctx, stmt, toValue)
	default:
		return nil, true, fmterr.Errorf(ctx.File().FileSet(), stmt.Source(), "cannot range over %s", keyKind.String())
	}
}

func evalRangeStmt(ctx *context, stmt *ir.RangeStmt) ([]elements.Element, bool, error) {
	kind := stmt.X.Type().Kind()
	if ir.IsRangeOk(kind) {
		return evalRangeStmtInteger(ctx, stmt, kind)
	}
	if kind == ir.ArrayKind {
		return evalRangeStmtArray(ctx, stmt)
	}
	return nil, true, errors.Errorf("cannot range over %s", kind.String())
}

func evalIfStmt(ctx *context, stmt *ir.IfStmt) ([]elements.Element, bool, error) {
	ctx.pushBlockFrame()
	defer ctx.popFrame()

	if stmt.Init != nil {
		if _, _, err := evalStmt(ctx, stmt.Init); err != nil {
			return nil, true, err
		}
	}
	condValue, err := evalAtom[bool](ctx, stmt.Cond)
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
		cNode, err := ctx.evalExpr(asg.X)
		if err != nil {
			return err
		}
		if cNode == nil {
			continue
		}
		if err := ctx.set(stmt.Src.Tok, asg.Storage, cNode); err != nil {
			return err
		}
	}
	return nil
}

func evalAssignCallStmt(ctx *context, stmt *ir.AssignCallStmt) error {
	nodes, err := evalCall(ctx, stmt.Call)
	if err != nil {
		return err
	}
	for i, dest := range stmt.List {
		node := nodes[i]
		if node == nil {
			continue
		}
		if err := ctx.set(stmt.Src.Tok, dest.Storage, node); err != nil {
			return err
		}
	}
	return nil
}

func unpackIfTuple(el elements.Element) []elements.Element {
	tpl, ok := el.(*elements.Tuple)
	if !ok {
		return []elements.Element{el}
	}
	return tpl.Elements()
}

func evalReturnStmt(ctx *context, ret *ir.ReturnStmt) ([]elements.Element, bool, error) {
	if len(ret.Results) == 0 {
		// Naked return.
		fields := ctx.currentFrame().owner.function.FuncType().Results.Fields()
		nodes := make([]elements.Element, len(fields))
		for i, field := range fields {
			var err error
			nodes[i], err = ctx.find(field.Name)
			if err != nil {
				return nil, false, err
			}
		}
		return nodes, true, nil
	}
	var returns []elements.Element
	for _, expr := range ret.Results {
		exprI, err := ctx.evalExpr(expr)
		if err != nil {
			return nil, false, err
		}
		returns = append(returns, unpackIfTuple(exprI)...)
	}
	return returns, true, nil
}

func evalValueRef(ctx *context, ref *ir.ValueRef) (elements.Element, error) {
	cNode, err := ctx.find(ref.Src)
	if err != nil {
		return nil, err
	}
	return cNode, nil
}

func evalCastToScalarExpr(ctx *context, expr ir.TypeCastExpr, x elements.NumericalElement, targetType ir.ArrayType) (elements.Element, error) {
	if targetType.Rank().NumAxes() > 0 {
		var err error
		x, err = ctx.eval.evaluator.ArrayOps().Reshape(elements.NewExprAt(ctx.File(), expr), x, nil)
		if err != nil {
			return nil, err
		}
	}
	return x.Cast(ctx, expr, targetType)
}

func evalArrayAxes(ctx *context, src ir.SourceNode, typ ir.ArrayType) ([]elements.NumericalElement, error) {
	rank, err := rankOf(ctx, src, typ)
	if err != nil {
		return nil, err
	}
	axes := make([]elements.NumericalElement, rank.NumAxes())
	for i, axis := range rank.Axes() {
		var err error
		axes[i], err = evalNumExpr(ctx, axis)
		if err != nil {
			return nil, err
		}
	}
	return axes, nil
}

func evalCastToArrayExpr(ctx *context, expr ir.TypeCastExpr, x elements.NumericalElement, targetType ir.ArrayType) (elements.Element, error) {
	origType := expr.Orig().Type()
	_, xDType := ir.Shape(origType)
	targetDType := targetType.DataType()
	targetKind := targetDType.Kind().DType()
	if xDType.Kind().DType() != targetKind {
		var err error
		x, err = x.Cast(ctx, expr, targetDType)
		if err != nil {
			return nil, err
		}
	}
	axes, err := evalArrayAxes(ctx, expr, targetType)
	if err != nil {
		return nil, err
	}
	reshape, err := ctx.eval.evaluator.ArrayOps().Reshape(elements.NewExprAt(ctx.File(), expr), x, axes)
	if err != nil {
		return nil, fmterr.Position(ctx.File().FileSet(), expr.Source(), err)
	}
	sourceType, ok := origType.(ir.ArrayType)
	if !ok {
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Source(), "cannot cast %T to %s", origType, reflect.TypeFor[ir.ArrayType]().Name())
	}
	if sourceType.DataType().Kind().DType() == targetKind {
		return reshape, nil
	}
	return reshape.Cast(ctx, expr, targetDType)
}

func evalCastExpr(ctx *context, expr ir.TypeCastExpr) (elements.Element, error) {
	x, err := ctx.evalExpr(expr.Orig())
	if err != nil {
		return nil, err
	}
	target := expr.Type()
	if named, ok := target.(*ir.NamedType); ok {
		recv, ok := x.(elements.Copier)
		if !ok {
			return nil, errors.Errorf("element %T cannot be copied", x)
		}
		return elements.NewNamedType(ctx.eval.evaluator.NewFunc, named, recv), nil
	}
	arrayType, ok := target.(ir.ArrayType)
	if !ok {
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Source(), "cast to %s not supported", target.String())
	}
	xNum, ok := x.(elements.NumericalElement)
	if !ok {
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Source(), "cannot cast element of type %T to %s", x, reflect.TypeFor[elements.NumericalElement]().Name())
	}
	if arrayType.Rank().IsAtomic() {
		return evalCastToScalarExpr(ctx, expr, xNum, arrayType)
	}
	return evalCastToArrayExpr(ctx, expr, xNum, arrayType)
}

func evalUnaryExpression(ctx *context, expr *ir.UnaryExpr) (elements.Element, error) {
	x, err := evalNumExpr(ctx, expr.X)
	if err != nil {
		return nil, err
	}
	if ir.IsStatic(expr.Type()) {
		return nil, errors.Errorf("not supported")
	}
	return x.UnaryOp(ctx, expr)
}

func evalBinaryExpression(ctx *context, expr *ir.BinaryExpr) (elements.Element, error) {
	x, err := evalNumExpr(ctx, expr.X)
	if err != nil {
		return nil, err
	}
	y, err := evalNumExpr(ctx, expr.Y)
	if err != nil {
		return nil, err
	}
	return x.BinaryOp(ctx, expr, x, y)
}

func evalStructLiteral(ctx *context, expr *ir.StructLitExpr) (elements.Element, error) {
	under := ir.Underlying(expr.Typ)
	structType, ok := under.(*ir.StructType)
	if !ok {
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Source(), "underlying type %T is not a structure", expr.Typ)
	}
	fields := make(map[string]elements.Element, structType.NumFields())
	for _, fieldLit := range expr.Elts {
		node, err := ctx.evalExpr(fieldLit.X)
		if err != nil {
			return nil, err
		}
		fields[fieldLit.Field.Name.Name] = node
	}
	strct := elements.NewStruct(structType, elements.NewValueAt(ctx.File(), expr), fields)
	nType, ok := expr.Typ.(*ir.NamedType)
	if !ok {
		return strct, nil
	}
	return elements.NewNamedType(ctx.eval.evaluator.NewFunc, nType, strct), nil
}

func evalSliceLiteral(ctx *context, expr *ir.SliceLitExpr) (elements.Element, error) {
	els := make([]elements.Element, len(expr.Elts))
	for i, expr := range expr.Elts {
		elt, err := ctx.evalExpr(expr)
		if err != nil {
			return nil, err
		}
		els[i] = elt
	}
	return elements.NewSlice(elements.NewExprAt(ctx.File(), expr), els), nil
}

// evalExpr evaluates an expression within the context.
func (ctx *context) evalExpr(expr ir.Expr) (elements.Element, error) {
	if expr == nil {
		return nil, errors.Errorf("cannot evaluate a nil expression")
	}
	switch exprT := expr.(type) {
	case *ir.ArrayLitExpr:
		return evalArrayLiteral(ctx, exprT)
	case *ir.AxisExpr:
		return ctx.evalExpr(exprT.X)
	case *ir.AxisInfer:
		return ctx.evalExpr(exprT.X)
	case *ir.NumberCastExpr:
		return evalNumberCastExpr(ctx, exprT)
	case *ir.SliceLitExpr:
		return evalSliceLiteral(ctx, exprT)
	case *ir.StructLitExpr:
		return evalStructLiteral(ctx, exprT)
	case *ir.CastExpr:
		return evalCastExpr(ctx, exprT)
	case *ir.TypeAssertExpr:
		return evalCastExpr(ctx, exprT)
	case *ir.CallExpr:
		return evalCallExpr(ctx, exprT)
	case *ir.UnaryExpr:
		return evalUnaryExpression(ctx, exprT)
	case *ir.ParenExpr:
		return ctx.evalExpr(exprT.X)
	case *ir.BinaryExpr:
		return evalBinaryExpression(ctx, exprT)
	case *ir.ValueRef:
		return evalValueRef(ctx, exprT)
	case *ir.SelectorExpr:
		return evalSelectorExpr(ctx, exprT)
	case *ir.FuncLit:
		return evalFuncLit(ctx, exprT)
	case *ir.IndexExpr:
		return evalIndexExpr(ctx, exprT)
	case *ir.EinsumExpr:
		return evalEinsumExpr(ctx, exprT)
	case *ir.StringLiteral:
		return elements.NewString(exprT)
	case *ir.NumberFloat:
		return numbers.NewFloat(elements.NewExprAt(ctx.File(), exprT), exprT.Val), nil
	case *ir.NumberInt:
		return numbers.NewInt(elements.NewExprAt(ctx.File(), exprT), exprT.Val), nil
	case ir.AtomicValue:
		return evalAtomicValue(ctx, exprT)
	case *ir.PackageRef:
		return ctx.find(exprT.X.Src)
	case *ir.AxisGroup:
		return ctx.find(&ast.Ident{NamePos: exprT.Src.NamePos, Name: exprT.Name})
	case *ir.FuncValExpr:
		return ctx.evalExpr(exprT.X)
	default:
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Source(), "cannot evaluate GX expression: %T not supported", expr)
	}
}

func evalNumExpr(ctx *context, expr ir.Expr) (elements.NumericalElement, error) {
	el, err := ctx.evalExpr(expr)
	if err != nil {
		return nil, err
	}
	numEl, ok := el.(elements.NumericalElement)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", el, reflect.TypeFor[elements.NumericalElement]())
	}
	return numEl, nil
}

func evalNumberCastExpr(ctx *context, expr *ir.NumberCastExpr) (elements.NumericalElement, error) {
	number, err := evalNumExpr(ctx, expr.X)
	if err != nil {
		return nil, err
	}
	return number.Cast(ctx, expr, expr.Typ)
}

func evalSelectorExpr(ctx *context, ref *ir.SelectorExpr) (elements.Element, error) {
	node, err := ctx.evalExpr(ref.X)
	if err != nil {
		return nil, err
	}
	slt, ok := node.(elements.Selector)
	if !ok {
		return nil, fmterr.Internalf(ctx.File().FileSet(), ref.Source(), "%T does not implement %s: cannot fetch member %s", node, reflect.TypeFor[elements.Selector](), ref.Src.Sel.Name)
	}
	return slt.Select(elements.NewNodeAt(ctx.File(), ref))
}

func evalFuncLit(ctx *context, ref *ir.FuncLit) (elements.Element, error) {
	return ctx.eval.evaluator.NewFunc(ref, nil), nil
}

func evalIndexExpr(ctx *context, ref *ir.IndexExpr) (elements.Element, error) {
	x, err := ctx.evalExpr(ref.X)
	if err != nil {
		return nil, err
	}
	slicer, ok := x.(elements.Slicer)
	if !ok {
		return nil, fmterr.Errorf(ctx.File().FileSet(), ref.Source(), "cannot index over %T", x)
	}
	index, err := evalNumExpr(ctx, ref.Index)
	if err != nil {
		return nil, err
	}
	return slicer.Slice(ctx, ref, index)
}

func evalEinsumExpr(ctx *context, ref *ir.EinsumExpr) (elements.Element, error) {
	x, err := evalNumExpr(ctx, ref.X)
	if err != nil {
		return nil, err
	}
	y, err := evalNumExpr(ctx, ref.Y)
	if err != nil {
		return nil, err
	}
	return ctx.eval.evaluator.ArrayOps().Einsum(elements.NewNodeAt(ctx.File(), ref), x, y)
}
