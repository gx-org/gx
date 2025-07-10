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
	"go/token"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/numbers"
)

func evalBlockStmt(ctx *context.Context, body *ir.BlockStmt) ([]ir.Element, bool, error) {
	var outs []ir.Element
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

func evalStmt(ctx *context.Context, node ir.Stmt) ([]ir.Element, bool, error) {
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
		return nil, false, fmterr.Errorf(ctx.File().FileSet(), node.Source(), "cannot evaluate GX node: %T not supported", node)
	}
}

func evalRangeForLoopOverInteger[T dtype.AlgebraType](ctx *context.Context, stmt *ir.RangeStmt, toValue valuer) ([]ir.Element, bool, error) {
	toValueT := toValue.(valuerT[T])
	indexType := ir.TypeFromKind(toValueT.kind)
	val, err := evalAtom[T](ctx, stmt.X)
	if err != nil {
		return nil, true, err
	}
	ctx.PushBlockFrame()
	defer ctx.PopFrame()
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
		iElement, err := ctx.Evaluator().ElementFromAtom(ctx, iExpr, iValue)
		if err != nil {
			return nil, true, err
		}
		if err := set(ctx, stmt.Src.Tok, stmt.Key, iElement); err != nil {
			return nil, false, err
		}
		element, stop, err := evalBlockStmt(ctx, stmt.Body)
		if stop || err != nil {
			return element, stop, err
		}
	}
	return nil, false, nil
}

func evalRangeStmtInteger(ctx *context.Context, stmt *ir.RangeStmt, xKind ir.Kind) ([]ir.Element, bool, error) {
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

func evalRangeStmtForLoopOverArray[T dtype.AlgebraType](ctx *context.Context, stmt *ir.RangeStmt, toValue valuer) ([]ir.Element, bool, error) {
	toValueT := toValue.(valuerT[T])
	indexType := ir.TypeFromKind(toValueT.kind)
	x, err := evalExpr(ctx, stmt.X)
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
		iElement, err := ctx.Evaluator().ElementFromAtom(ctx, iExpr, iValue)
		if err != nil {
			return nil, false, err
		}
		if err := set(ctx, stmt.Src.Tok, stmt.Key, iElement); err != nil {
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
			reshapedElement, err := elementI.Reshape(ctx, valueExpr, dims)
			if err != nil {
				return nil, false, err
			}
			if err := set(ctx, stmt.Src.Tok, stmt.Value, reshapedElement); err != nil {
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

func evalRangeStmtArray(ctx *context.Context, stmt *ir.RangeStmt) ([]ir.Element, bool, error) {
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

func evalRangeStmt(ctx *context.Context, stmt *ir.RangeStmt) ([]ir.Element, bool, error) {
	kind := stmt.X.Type().Kind()
	if ir.IsRangeOk(kind) {
		return evalRangeStmtInteger(ctx, stmt, kind)
	}
	if kind == ir.ArrayKind {
		return evalRangeStmtArray(ctx, stmt)
	}
	return nil, true, errors.Errorf("cannot range over %s", kind.String())
}

func evalIfStmt(ctx *context.Context, stmt *ir.IfStmt) ([]ir.Element, bool, error) {
	ctx.PushBlockFrame()
	defer ctx.PopFrame()

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

func evalAssignExprStmt(ctx *context.Context, stmt *ir.AssignExprStmt) error {
	for _, asg := range stmt.List {
		cNode, err := evalExpr(ctx, asg.X)
		if err != nil {
			return err
		}
		if cNode == nil {
			continue
		}
		if err := set(ctx, stmt.Src.Tok, asg.Storage, cNode); err != nil {
			return err
		}
	}
	return nil
}

func evalAssignCallStmt(ctx *context.Context, stmt *ir.AssignCallStmt) error {
	nodes, err := evalCall(ctx, stmt.Call)
	if err != nil {
		return err
	}
	for i, dest := range stmt.List {
		node := nodes[i]
		if node == nil {
			continue
		}
		if err := set(ctx, stmt.Src.Tok, dest.Storage, node); err != nil {
			return err
		}
	}
	return nil
}

func unpackIfTuple(el ir.Element) []ir.Element {
	tpl, ok := el.(*elements.Tuple)
	if !ok {
		return []ir.Element{el}
	}
	return tpl.Elements()
}

func evalReturnStmt(ctx *context.Context, ret *ir.ReturnStmt) ([]ir.Element, bool, error) {
	if len(ret.Results) == 0 {
		// Naked return.
		fields := ctx.CurrentFunc().FuncType().Results.Fields()
		nodes := make([]ir.Element, len(fields))
		fr := ctx.CurrentFrame()
		for i, field := range fields {
			var err error
			nodes[i], err = fr.Find(field.Name)
			if err != nil {
				return nil, false, err
			}
		}
		return nodes, true, nil
	}
	var returns []ir.Element
	for _, expr := range ret.Results {
		exprI, err := evalExpr(ctx, expr)
		if err != nil {
			return nil, false, err
		}
		returns = append(returns, unpackIfTuple(exprI)...)
	}
	return returns, true, nil
}

func evalCastToScalarExpr(ctx *context.Context, expr ir.TypeCastExpr, x evaluator.NumericalElement, targetType ir.ArrayType) (ir.Element, error) {
	if len(x.Shape().AxisLengths) > 0 {
		return x.Reshape(ctx, expr, nil)
	}
	return x.Cast(ctx, expr, targetType)
}

func evalArrayAxes(ctx *context.Context, src ir.SourceNode, typ ir.ArrayType) ([]evaluator.NumericalElement, error) {
	rank, err := rankOf(ctx, src, typ)
	if err != nil {
		return nil, err
	}
	axes := make([]evaluator.NumericalElement, len(rank.Axes()))
	for i, axis := range rank.Axes() {
		var err error
		axes[i], err = evalNumExpr(ctx, axis)
		if err != nil {
			return nil, err
		}
	}
	return axes, nil
}

var one, _ = values.AtomIntegerValue(ir.IntLenType(), ir.Int(1))

func evalCastAtomToArrayExpr(ctx *context.Context, expr ir.TypeCastExpr, x evaluator.NumericalElement, axes []evaluator.NumericalElement) (ir.Element, error) {
	srcExpr := elements.NewExprAt(ctx.File(), expr)
	arrayOps := ctx.Evaluator().ArrayOps()
	shapeOfOnes := make([]evaluator.NumericalElement, len(axes))
	for i := range axes {
		var err error
		shapeOfOnes[i], err = ctx.Evaluator().ElementFromAtom(ctx, expr, one)
		if err != nil {
			return nil, err
		}
	}
	reshaped, err := x.Reshape(ctx, srcExpr.ToExprAt().Node(), shapeOfOnes)
	if err != nil {
		return nil, err
	}
	return arrayOps.BroadcastInDim(ctx, expr, reshaped, axes)
}

func evalCastToArrayExpr(ctx *context.Context, expr ir.TypeCastExpr, x evaluator.NumericalElement, targetType ir.ArrayType) (ir.Element, error) {
	origType := expr.Orig().Type()
	origRank, xDType := ir.Shape(origType)
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
	if origRank.IsAtomic() {
		return evalCastAtomToArrayExpr(ctx, expr, x, axes)
	}
	reshape, err := x.Reshape(ctx, expr, axes)
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

func evalCastExpr(ctx *context.Context, expr ir.TypeCastExpr) (ir.Element, error) {
	x, err := evalExpr(ctx, expr.Orig())
	if err != nil {
		return nil, err
	}
	target := expr.Type()
	if named, ok := target.(*ir.NamedType); ok {
		recv, ok := x.(elements.Copier)
		if !ok {
			return nil, errors.Errorf("element %T cannot be copied", x)
		}
		return elements.NewNamedType(ctx.NewFunc, named, recv), nil
	}
	arrayType, ok := target.(ir.ArrayType)
	if !ok {
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Source(), "cast to %s not supported", target.String())
	}
	xNum, ok := elements.Underlying(x).(evaluator.NumericalElement)
	if !ok {
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Source(), "cannot cast element of type %T to %s", x, reflect.TypeFor[evaluator.NumericalElement]().Name())
	}
	if arrayType.Rank().IsAtomic() {
		return evalCastToScalarExpr(ctx, expr, xNum, arrayType)
	}
	return evalCastToArrayExpr(ctx, expr, xNum, arrayType)
}

func evalUnaryExpression(ctx *context.Context, expr *ir.UnaryExpr) (ir.Element, error) {
	x, err := evalNumExpr(ctx, expr.X)
	if err != nil {
		return nil, err
	}
	if ir.IsStatic(expr.Type()) {
		return nil, errors.Errorf("not supported")
	}
	return x.UnaryOp(ctx, expr)
}

func evalBinaryExpression(ctx *context.Context, expr *ir.BinaryExpr) (ir.Element, error) {
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

func evalStructLiteral(ctx *context.Context, expr *ir.StructLitExpr) (ir.Element, error) {
	under := ir.Underlying(expr.Typ)
	structType, ok := under.(*ir.StructType)
	if !ok {
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Source(), "underlying type %T is not a structure", expr.Typ)
	}
	fields := make(map[string]ir.Element, structType.NumFields())
	for _, fieldLit := range expr.Elts {
		node, err := evalExpr(ctx, fieldLit.X)
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
	return elements.NewNamedType(ctx.NewFunc, nType, strct), nil
}

func evalSliceLiteral(ctx *context.Context, expr *ir.SliceLitExpr) (ir.Element, error) {
	els := make([]ir.Element, len(expr.Elts))
	for i, expr := range expr.Elts {
		elt, err := evalExpr(ctx, expr)
		if err != nil {
			return nil, err
		}
		els[i] = elt
	}
	return elements.NewSlice(expr.Type(), els), nil
}

// evalExpr evaluates an expression within the Context.
func evalExpr(ctx *context.Context, expr ir.Expr) (ir.Element, error) {
	if expr == nil {
		return nil, errors.Errorf("cannot evaluate a nil expression")
	}
	switch exprT := expr.(type) {
	case *ir.ArrayLitExpr:
		return evalArrayLiteral(ctx, exprT)
	case *ir.AxisExpr:
		return evalExpr(ctx, exprT.X)
	case *ir.AxisInfer:
		return evalExpr(ctx, exprT.X)
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
		return evalExpr(ctx, exprT.X)
	case *ir.BinaryExpr:
		return evalBinaryExpression(ctx, exprT)
	case *ir.ValueRef:
		return ctx.CurrentFrame().Find(exprT.Src)
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
		return ctx.CurrentFrame().Find(exprT.X.Src)
	case *ir.FuncValExpr:
		return evalExpr(ctx, exprT.X)
	default:
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Source(), "cannot evaluate GX expression: %T not supported", expr)
	}
}

func evalNumExpr(ctx *context.Context, expr ir.Expr) (evaluator.NumericalElement, error) {
	el, err := evalExpr(ctx, expr)
	if err != nil {
		return nil, err
	}
	numEl, ok := elements.Underlying(el).(evaluator.NumericalElement)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", el, reflect.TypeFor[evaluator.NumericalElement]())
	}
	return numEl, nil
}

func evalNumberCastExpr(ctx *context.Context, expr *ir.NumberCastExpr) (evaluator.NumericalElement, error) {
	number, err := evalNumExpr(ctx, expr.X)
	if err != nil {
		return nil, err
	}
	return number.Cast(ctx, expr, expr.Typ)
}

func evalSelectorExpr(ctx *context.Context, ref *ir.SelectorExpr) (ir.Element, error) {
	node, err := evalExpr(ctx, ref.X)
	if err != nil {
		return nil, err
	}
	slt, ok := node.(elements.Selector)
	if !ok {
		return nil, fmterr.Internalf(ctx.File().FileSet(), ref.Source(), "%T does not implement %s: cannot fetch member %s", node, reflect.TypeFor[elements.Selector](), ref.Src.Sel.Name)
	}
	return slt.Select(elements.NewNodeAt(ctx.File(), ref))
}

func evalFuncLit(ctx *context.Context, ref *ir.FuncLit) (ir.Element, error) {
	return ctx.NewFunc(ref, nil), nil
}

func evalIndexExpr(ctx *context.Context, ref *ir.IndexExpr) (ir.Element, error) {
	x, err := evalExpr(ctx, ref.X)
	if err != nil {
		return nil, err
	}
	x = elements.Underlying(x)
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

func evalEinsumExpr(ctx *context.Context, ref *ir.EinsumExpr) (ir.Element, error) {
	x, err := evalNumExpr(ctx, ref.X)
	if err != nil {
		return nil, err
	}
	y, err := evalNumExpr(ctx, ref.Y)
	if err != nil {
		return nil, err
	}
	return ctx.Evaluator().ArrayOps().Einsum(ctx, ref, x, y)
}

func evalAtom[T dtype.GoDataType](ctx *context.Context, expr ir.Expr) (val T, err error) {
	el, err := evalExpr(ctx, expr)
	if err != nil {
		var zero T
		return zero, err
	}
	return elements.ConstantScalarFromElement[T](el)
}

func evalCallExpr(ctx *context.Context, expr *ir.CallExpr) (ir.Element, error) {
	outs, err := evalCall(ctx, expr)
	if err != nil {
		return nil, err
	}
	return ToSingleElement(ctx, expr, outs)
}

func evalCall(ctx *context.Context, expr *ir.CallExpr) ([]ir.Element, error) {
	// Fetch the function and check that it is callable.
	fnNode, err := ctx.EvalExpr(expr.Callee.X)
	if err != nil {
		return nil, err
	}
	fn, ok := fnNode.(elements.Func)
	if !ok {
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Source(), "%T is not callable", fnNode)
	}

	// Evaluate the arguments to pass to the function.
	args := make([]ir.Element, len(expr.Args))
	for i, arg := range expr.Args {
		el, err := ctx.EvalExpr(arg)
		if err != nil {
			return nil, err
		}
		args[i] = el
	}
	return fn.Call(ctx, expr, args)
}

func set(ctx *context.Context, tok token.Token, dest ir.Storage, value ir.Element) error {
	switch destT := dest.(type) {
	case *ir.LocalVarStorage:
		if !ir.ValidIdent(destT.Src) {
			return nil
		}
		if tok == token.ILLEGAL {
			return nil
		}
		if tok == token.DEFINE {
			ctx.CurrentFrame().Define(destT.Src.Name, value)
			return nil
		}
		return ctx.CurrentFrame().Assign(destT.Src.Name, value)
	case *ir.StructFieldStorage:
		receiver, err := evalExpr(ctx, destT.Sel.X)
		if err != nil {
			return err
		}
		strt, ok := elements.Underlying(receiver).(*elements.Struct)
		if !ok {
			return fmterr.Errorf(ctx.File().FileSet(), dest.Source(), "cannot convert %T to %T", receiver, strt)
		}
		strt.SetField(destT.Sel.Src.Sel.Name, value)
		return nil
	case *ir.FieldStorage:
		return ctx.CurrentFrame().Assign(destT.Field.Name.Name, value)
	case *ir.AssignExpr:
		return ctx.CurrentFrame().Assign(destT.NameDef().Name, value)
	default:
		return fmterr.Errorf(ctx.File().FileSet(), dest.Source(), "cannot assign %v to %T: not supported", value, destT)
	}
}

// ToSingleElement packs multiple elements into a tuple.
// If the slice els contains only one element, this element is returned.
func ToSingleElement(ctx ir.Evaluator, node ir.SourceNode, els []ir.Element) (ir.Element, error) {
	switch len(els) {
	case 0:
		return nil, nil
	case 1:
		return els[0], nil
	default:
		return elements.NewTuple(els), nil
	}

}
