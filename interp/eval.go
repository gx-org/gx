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
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp/numbers"
)

func evalBlockStmt(ctx *FileScope, body *ir.BlockStmt) ([]ir.Element, bool, error) {
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

func evalStmt(fitp *FileScope, node ir.Stmt) ([]ir.Element, bool, error) {
	switch nodeT := node.(type) {
	case *ir.AssignCallStmt:
		return nil, false, evalAssignCallStmt(fitp, nodeT)
	case *ir.AssignExprStmt:
		return nil, false, evalAssignExprStmt(fitp, nodeT)
	case *ir.RangeStmt:
		return evalRangeStmt(fitp, nodeT)
	case *ir.IfStmt:
		return evalIfStmt(fitp, nodeT)
	case *ir.ReturnStmt:
		return evalReturnStmt(fitp, nodeT)
	case *ir.BlockStmt:
		return evalBlockStmt(fitp, nodeT)
	case *ir.ExprStmt:
		_, err := evalExpr(fitp, nodeT.X)
		return nil, false, err
	default:
		return nil, false, fmterr.Errorf(fitp.File().FileSet(), node.Node(), "cannot evaluate GX node: %T not supported", node)
	}
}

func evalRangeForLoopOverInteger[T dtype.AlgebraType](fitp *FileScope, stmt *ir.RangeStmt, toValue valuer) ([]ir.Element, bool, error) {
	toValueT := toValue.(valuerT[T])
	indexType := ir.TypeFromKind(toValueT.kind)
	val, err := evalAtom[T](fitp, stmt.X)
	if err != nil {
		return nil, true, err
	}
	fitp.ctx.PushBlockFrame()
	defer fitp.ctx.PopFrame()
	for i := T(0); i < val; i++ {
		iExpr := &ir.AtomicValueT[T]{
			Src: stmt.Key.Node().(ast.Expr),
			Val: i,
			Typ: indexType,
		}
		iValue, err := toValueT.toAtomValue(iExpr.Type(), i)
		if err != nil {
			return nil, true, err
		}
		iElement, err := fitp.Evaluator().ElementFromAtom(fitp.File(), iExpr, iValue)
		if err != nil {
			return nil, true, err
		}
		if err := set(fitp, stmt.Src.Tok, stmt.Key, iElement); err != nil {
			return nil, false, err
		}
		element, stop, err := evalBlockStmt(fitp, stmt.Body)
		if stop || err != nil {
			return element, stop, err
		}
	}
	return nil, false, nil
}

func evalRangeStmtInteger(fitp *FileScope, stmt *ir.RangeStmt, xKind irkind.Kind) ([]ir.Element, bool, error) {
	toValue, err := newValuer(fitp, stmt.X, xKind)
	if err != nil {
		return nil, false, err
	}
	switch xKind {
	case irkind.IntLen:
		return evalRangeForLoopOverInteger[ir.Int](fitp, stmt, toValue)
	default:
		return nil, true, fmterr.Errorf(fitp.File().FileSet(), stmt.Node(), "cannot range over %s", xKind.String())
	}
}

func evalRangeStmtForLoopOverArray[T dtype.AlgebraType](fitp *FileScope, stmt *ir.RangeStmt, toValue valuer) ([]ir.Element, bool, error) {
	toValueT := toValue.(valuerT[T])
	indexType := ir.TypeFromKind(toValueT.kind)
	x, err := evalExpr(fitp, stmt.X)
	if err != nil {
		return nil, false, err
	}
	value, ok := x.(elements.ArraySlicer)
	if !ok {
		return nil, false, fmterr.Errorf(fitp.File().FileSet(), stmt.Node(), "cannot range over %T", x)
	}
	arrayShape, err := elements.ShapeFromElement(value)
	if err != nil {
		return nil, false, fmterr.AtNode(fitp.File().FileSet(), stmt.Node(), err)
	}
	for i := 0; i < arrayShape.AxisLengths[0]; i++ {
		iExpr := &ir.AtomicValueT[T]{
			Src: stmt.Key.Node().(ast.Expr),
			Val: T(i),
			Typ: indexType,
		}
		iValue, err := toValueT.toAtomValue(iExpr.Type(), T(i))
		if err != nil {
			return nil, false, err
		}
		iElement, err := fitp.Evaluator().ElementFromAtom(fitp.File(), iExpr, iValue)
		if err != nil {
			return nil, false, err
		}
		if err := set(fitp, stmt.Src.Tok, stmt.Key, iElement); err != nil {
			return nil, false, err
		}
		if stmt.Value != nil {
			valueExpr := &ir.SliceLitExpr{
				Src: stmt.Src.X,
				Typ: value.Type(), // TODO(396633820): compute the correct type (the dtype will be correct but not the shape)
			}
			elementI, err := value.SliceArray(valueExpr, iElement)
			if err != nil {
				return nil, false, err
			}
			dims, err := dimsAsElements(fitp, valueExpr, arrayShape.AxisLengths[1:])
			if err != nil {
				return nil, false, err
			}
			reshapedElement, err := elementI.Reshape(fitp.env, valueExpr, dims)
			if err != nil {
				return nil, false, err
			}
			if err := set(fitp, stmt.Src.Tok, stmt.Value, reshapedElement); err != nil {
				return nil, false, err
			}
		}
		outs, stop, err := evalBlockStmt(fitp, stmt.Body)
		if stop || err != nil {
			return outs, stop, err
		}
	}
	return nil, false, nil
}

func evalRangeStmtArray(fitp *FileScope, stmt *ir.RangeStmt) ([]ir.Element, bool, error) {
	keyKind := stmt.Key.Type().Kind()
	toValue, err := newValuer(fitp, stmt.X, keyKind)
	if err != nil {
		return nil, false, err
	}
	switch keyKind {
	case irkind.Int64:
		return evalRangeStmtForLoopOverArray[ir.Int](fitp, stmt, toValue)
	default:
		return nil, true, fmterr.Errorf(fitp.File().FileSet(), stmt.Node(), "cannot range over %s", keyKind.String())
	}
}

func evalRangeStmt(fitp *FileScope, stmt *ir.RangeStmt) ([]ir.Element, bool, error) {
	kind := stmt.X.Type().Kind()
	if irkind.IsRangeOk(kind) {
		return evalRangeStmtInteger(fitp, stmt, kind)
	}
	if kind == irkind.Array {
		return evalRangeStmtArray(fitp, stmt)
	}
	return nil, true, errors.Errorf("cannot range over %s", kind.String())
}

func evalIfStmt(fitp *FileScope, stmt *ir.IfStmt) ([]ir.Element, bool, error) {
	fitp.Context().PushBlockFrame()
	defer fitp.Context().PopFrame()

	if stmt.Init != nil {
		if _, _, err := evalStmt(fitp, stmt.Init); err != nil {
			return nil, true, err
		}
	}
	condValue, err := evalAtom[bool](fitp, stmt.Cond)
	if err != nil {
		return nil, true, err
	}
	if condValue {
		return evalBlockStmt(fitp, stmt.Body)
	}
	if stmt.Else == nil {
		return nil, false, nil
	}
	return evalStmt(fitp, stmt.Else)
}

func evalAssignExprStmt(fitp *FileScope, stmt *ir.AssignExprStmt) error {
	for _, asg := range stmt.List {
		cNode, err := evalExpr(fitp, asg.X)
		if err != nil {
			return err
		}
		if cNode == nil {
			continue
		}
		if err := set(fitp, stmt.Src.Tok, asg.Storage, cNode); err != nil {
			return err
		}
	}
	return nil
}

func evalAssignCallStmt(fitp *FileScope, stmt *ir.AssignCallStmt) error {
	nodes, err := evalCall(fitp, stmt.Call.FuncCall())
	if err != nil {
		return err
	}
	for i, dest := range stmt.List {
		node := nodes[i]
		if node == nil {
			continue
		}
		if err := set(fitp, stmt.Src.Tok, dest.Storage, node); err != nil {
			return err
		}
	}
	return nil
}

func unpackIfTuple(el ir.Element) []ir.Element {
	tpl, ok := el.(*Tuple)
	if !ok {
		return []ir.Element{el}
	}
	return tpl.Elements()
}

func evalReturnStmt(fitp *FileScope, ret *ir.ReturnStmt) ([]ir.Element, bool, error) {
	if len(ret.Results) == 0 {
		// Naked return.
		fields := fitp.ctx.CurrentFunc().FuncType().Results.Fields()
		nodes := make([]ir.Element, len(fields))
		fr := fitp.ctx.CurrentFrame()
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
		exprI, err := evalExpr(fitp, expr)
		if err != nil {
			return nil, false, err
		}
		returns = append(returns, unpackIfTuple(exprI)...)
	}
	return returns, true, nil
}

func evalCastToScalarExpr(fitp *FileScope, expr ir.TypeCastExpr, x evaluator.NumericalElement, targetType ir.ArrayType) (ir.Element, error) {
	xShape, err := elements.ShapeFromElement(x)
	if err != nil {
		return nil, fmterr.AtNode(fitp.File().FileSet(), expr.Node(), err)
	}
	if len(xShape.AxisLengths) > 0 {
		return x.Reshape(fitp.env, expr, nil)
	}
	return x.Cast(fitp.env, expr, targetType)
}

func evalArrayAxes(fitp *FileScope, src ir.Node, typ ir.ArrayType) ([]evaluator.NumericalElement, error) {
	rank, err := rankOf(fitp.env, src, typ)
	if err != nil {
		return nil, err
	}
	axes := make([]evaluator.NumericalElement, len(rank.Axes()))
	for i, axis := range rank.Axes() {
		var err error
		axes[i], err = evalNumExpr(fitp, axis)
		if err != nil {
			return nil, err
		}
	}
	return axes, nil
}

var one, _ = values.AtomIntegerValue(ir.IntLenType(), ir.Int(1))

func evalCastAtomToArrayExpr(fitp *FileScope, expr ir.TypeCastExpr, x evaluator.NumericalElement, axes []evaluator.NumericalElement) (ir.Element, error) {
	srcExpr := elements.NewExprAt(fitp.File(), expr)
	arrayOps := fitp.Evaluator().ArrayOps()
	shapeOfOnes := make([]evaluator.NumericalElement, len(axes))
	for i := range axes {
		var err error
		shapeOfOnes[i], err = fitp.Evaluator().ElementFromAtom(fitp.File(), expr, one)
		if err != nil {
			return nil, err
		}
	}
	reshaped, err := x.Reshape(fitp.env, srcExpr.ToExprAt().Node(), shapeOfOnes)
	if err != nil {
		return nil, err
	}
	return arrayOps.BroadcastInDim(fitp, expr, reshaped, axes)
}

func evalCastToArrayExpr(fitp *FileScope, expr ir.TypeCastExpr, x evaluator.NumericalElement, targetType ir.ArrayType) (ir.Element, error) {
	origType := expr.Orig().Type()
	origRank, xDType := ir.Shape(origType)
	targetDType := targetType.DataType()
	targetKind := targetDType.Kind().DType()
	if xDType.Kind().DType() != targetKind {
		var err error
		x, err = x.Cast(fitp.env, expr, targetDType)
		if err != nil {
			return nil, err
		}
	}
	axes, err := evalArrayAxes(fitp, expr, targetType)
	if err != nil {
		return nil, err
	}
	if origRank.IsAtomic() {
		return evalCastAtomToArrayExpr(fitp, expr, x, axes)
	}
	reshape, err := x.Reshape(fitp.env, expr, axes)
	if err != nil {
		return nil, fmterr.AtNode(fitp.File().FileSet(), expr.Node(), err)
	}
	sourceType, ok := origType.(ir.ArrayType)
	if !ok {
		return nil, fmterr.Errorf(fitp.File().FileSet(), expr.Node(), "cannot cast %T to %s", origType, reflect.TypeFor[ir.ArrayType]().Name())
	}
	if sourceType.DataType().Kind().DType() == targetKind {
		return reshape, nil
	}
	return reshape.Cast(fitp.env, expr, targetDType)
}

func evalCastExpr(fitp *FileScope, expr ir.TypeCastExpr) (ir.Element, error) {
	x, err := evalExpr(fitp, expr.Orig())
	if err != nil {
		return nil, err
	}
	target := expr.Type()
	if named, ok := target.(*ir.NamedType); ok {
		recv, ok := x.(elements.Copier)
		if !ok {
			return nil, errors.Errorf("element %T cannot be copied", x)
		}
		return fun.NewNamedType(fitp.NewFunc, named, recv), nil
	}
	arrayType, ok := target.(ir.ArrayType)
	if !ok {
		return nil, fmterr.Errorf(fitp.File().FileSet(), expr.Node(), "cast to %s not supported", target.String())
	}
	xNum, ok := elements.Underlying(x).(evaluator.NumericalElement)
	if !ok {
		return nil, fmterr.Errorf(fitp.File().FileSet(), expr.Node(), "cannot cast element of type %T to %s", x, reflect.TypeFor[evaluator.NumericalElement]().Name())
	}
	if arrayType.Rank().IsAtomic() {
		return evalCastToScalarExpr(fitp, expr, xNum, arrayType)
	}
	return evalCastToArrayExpr(fitp, expr, xNum, arrayType)
}

func evalUnaryExpression(fitp *FileScope, expr *ir.UnaryExpr) (ir.Element, error) {
	x, err := evalNumExpr(fitp, expr.X)
	if err != nil {
		return nil, err
	}
	if ir.IsStatic(expr.Type()) {
		return nil, errors.Errorf("not supported")
	}
	return x.UnaryOp(fitp.env, expr)
}

func evalBinaryExpression(fitp *FileScope, expr *ir.BinaryExpr) (ir.Element, error) {
	x, err := evalNumExpr(fitp, expr.X)
	if err != nil {
		return nil, err
	}
	y, err := evalNumExpr(fitp, expr.Y)
	if err != nil {
		return nil, err
	}
	return x.BinaryOp(fitp.env, expr, x, y)
}

func evalStructLiteral(fitp *FileScope, expr *ir.StructLitExpr) (ir.Element, error) {
	under := ir.Underlying(expr.Typ)
	structType, ok := under.(*ir.StructType)
	if !ok {
		return nil, fmterr.Errorf(fitp.File().FileSet(), expr.Node(), "underlying type %T is not a structure", expr.Typ)
	}
	fields := make(map[string]ir.Element, structType.NumFields())
	for _, fieldLit := range expr.Elts {
		node, err := evalExpr(fitp, fieldLit.X)
		if err != nil {
			return nil, err
		}
		fields[fieldLit.Field.Name.Name] = node
	}
	strct := elements.NewStruct(structType, fields)
	nType, ok := expr.Typ.(*ir.NamedType)
	if !ok {
		return strct, nil
	}
	return fun.NewNamedType(fitp.NewFunc, nType, strct), nil
}

func evalSliceLiteral(fitp *FileScope, expr *ir.SliceLitExpr) (ir.Element, error) {
	els := make([]ir.Element, len(expr.Elts))
	for i, expr := range expr.Elts {
		elt, err := evalExpr(fitp, expr)
		if err != nil {
			return nil, err
		}
		els[i] = elt
	}
	return elements.NewSlice(expr.Type(), els), nil
}

// evalExpr evaluates an expression within the Context.
func evalExpr(fitp *FileScope, expr ir.Expr) (ir.Element, error) {
	if expr == nil {
		return nil, errors.Errorf("cannot evaluate a nil expression")
	}
	switch exprT := expr.(type) {
	case *ir.ArrayLitExpr:
		return evalArrayLiteral(fitp, exprT)
	case *ir.AxisExpr:
		return evalExpr(fitp, exprT.X)
	case *ir.AxisInfer:
		return evalExpr(fitp, exprT.X)
	case *ir.NumberCastExpr:
		return evalNumberCastExpr(fitp, exprT)
	case *ir.SliceLitExpr:
		return evalSliceLiteral(fitp, exprT)
	case *ir.StructLitExpr:
		return evalStructLiteral(fitp, exprT)
	case *ir.CastExpr:
		return evalCastExpr(fitp, exprT)
	case *ir.TypeAssertExpr:
		return evalCastExpr(fitp, exprT)
	case *ir.FuncCallExpr:
		return evalCallExpr(fitp, exprT)
	case *ir.UnaryExpr:
		return evalUnaryExpression(fitp, exprT)
	case *ir.ParenExpr:
		return evalExpr(fitp, exprT.X)
	case *ir.BinaryExpr:
		return evalBinaryExpression(fitp, exprT)
	case *ir.ValueRef:
		return evalValueRef(fitp, exprT)
	case *ir.SelectorExpr:
		return evalSelectorExpr(fitp, exprT)
	case *ir.FuncLit:
		return fitp.env.FuncEval().NewFuncLit(fitp.env, exprT)
	case *ir.MacroCallExpr:
		return fitp.env.FuncEval().NewFunc(exprT.F, nil), nil
	case *ir.IndexExpr:
		return evalIndexExpr(fitp, exprT)
	case *ir.EinsumExpr:
		return evalEinsumExpr(fitp, exprT)
	case *ir.StringLiteral:
		return elements.NewString(exprT)
	case *ir.NumberFloat:
		return numbers.NewFloat(elements.NewExprAt(fitp.File(), exprT), exprT.Val), nil
	case *ir.NumberInt:
		return numbers.NewInt(elements.NewExprAt(fitp.File(), exprT), exprT.Val), nil
	case ir.AtomicValue:
		return evalAtomicValue(fitp, exprT)
	case *ir.PackageRef:
		return fitp.ctx.CurrentFrame().Find(exprT.X.Src)
	case *ir.FuncValExpr:
		return evalFuncValExpr(fitp, exprT)
	case *ir.TypeValExpr:
		return exprT, nil
	default:
		return nil, fmterr.Errorf(fitp.File().FileSet(), expr.Node(), "cannot evaluate GX expression: %T not supported", expr)
	}
}

func evalFuncValExpr(fitp *FileScope, expr *ir.FuncValExpr) (ir.Element, error) {
	lit, isLit := expr.F.(*ir.FuncLit)
	if isLit {
		return fitp.env.FuncEval().NewFuncLit(fitp.env, lit)
	}
	return fitp.NewFunc(expr.F, nil), nil
}

func evalValueRef(fitp *FileScope, ref *ir.ValueRef) (ir.Element, error) {
	if ref.Src == nil {
		return ref.Stor, nil
	}
	return fitp.ctx.CurrentFrame().Find(ref.Src)
}

func evalNumExpr(fitp *FileScope, expr ir.Expr) (evaluator.NumericalElement, error) {
	el, err := evalExpr(fitp, expr)
	if err != nil {
		return nil, err
	}
	numEl, ok := elements.Underlying(el).(evaluator.NumericalElement)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", el, reflect.TypeFor[evaluator.NumericalElement]())
	}
	return numEl, nil
}

func evalNumberCastExpr(fitp *FileScope, expr *ir.NumberCastExpr) (evaluator.NumericalElement, error) {
	number, err := evalNumExpr(fitp, expr.X)
	if err != nil {
		return nil, err
	}
	tp := expr.Typ
	if tpParam, ok := expr.Typ.(*ir.TypeParam); ok {
		tpEl, err := fitp.ctx.CurrentFrame().Find(tpParam.Field.Name)
		if err != nil {
			return nil, err
		}
		tp, ok = elements.Underlying(tpEl).(ir.Type)
		if !ok {
			return nil, errors.Errorf("%T is not a type", tpEl)
		}
	}
	return number.Cast(fitp.env, expr, tp)
}

func evalSelectorExpr(fitp *FileScope, ref *ir.SelectorExpr) (ir.Element, error) {
	node, err := evalExpr(fitp, ref.X)
	if err != nil {
		return nil, err
	}
	slt, ok := node.(elements.Selector)
	if !ok {
		return nil, fmterr.Internalf(fitp.File().FileSet(), ref.Node(), "%T does not implement %s: cannot fetch member %s", node, reflect.TypeFor[elements.Selector](), ref.Src.Sel.Name)
	}
	return slt.Select(ref)
}

func evalIndexExpr(fitp *FileScope, ref *ir.IndexExpr) (ir.Element, error) {
	x, err := evalExpr(fitp, ref.X)
	if err != nil {
		return nil, err
	}
	x = elements.Underlying(x)
	slicer, ok := x.(elements.Slicer)
	if !ok {
		return nil, fmterr.Errorf(fitp.File().FileSet(), ref.Node(), "cannot index over %T", x)
	}
	index, err := evalNumExpr(fitp, ref.Index)
	if err != nil {
		return nil, err
	}
	return slicer.Slice(ref, index)
}

func evalEinsumExpr(fitp *FileScope, ref *ir.EinsumExpr) (ir.Element, error) {
	x, err := evalNumExpr(fitp, ref.X)
	if err != nil {
		return nil, err
	}
	y, err := evalNumExpr(fitp, ref.Y)
	if err != nil {
		return nil, err
	}
	return fitp.Evaluator().ArrayOps().Einsum(fitp, ref, x, y)
}

func evalAtom[T dtype.GoDataType](fitp *FileScope, expr ir.Expr) (val T, err error) {
	el, err := evalExpr(fitp, expr)
	if err != nil {
		var zero T
		return zero, err
	}
	return elements.ConstantScalarFromElement[T](el)
}

func evalCallee(fitp *FileScope, callee ir.Callee) (fun.Func, error) {
	switch calleeT := callee.(type) {
	case *ir.FuncValExpr:
		fnNode, err := fitp.EvalExpr(calleeT.X)
		if err != nil {
			return nil, err
		}
		fn, ok := fnNode.(fun.Func)
		if !ok {
			return nil, fmterr.Errorf(fitp.File().FileSet(), callee.Node(), "%T is not callable", fnNode)
		}
		return fn, nil
	case *ir.MacroCallExpr:
		return fitp.NewFunc(calleeT.Func(), nil), nil
	default:
		return nil, errors.Errorf("callee type %T not supported", callee)
	}
}

func evalCallExpr(fitp *FileScope, expr *ir.FuncCallExpr) (ir.Element, error) {
	outs, err := evalCall(fitp, expr)
	if err != nil {
		return nil, err
	}
	return ToSingleElement(fitp, expr, outs)
}

func evalCall(fitp *FileScope, expr *ir.FuncCallExpr) ([]ir.Element, error) {
	callee, err := evalCallee(fitp, expr.Callee)
	if err != nil {
		return nil, err
	}
	// Evaluate the arguments to pass to the function.
	args := make([]ir.Element, len(expr.Args))
	for i, arg := range expr.Args {
		el, err := fitp.EvalExpr(arg)
		if err != nil {
			return nil, err
		}
		args[i] = el
	}
	return callee.Call(fitp.env, expr, args)
}

func evalFuncCall(fitp *FileScope, fn fun.Func, expr *ir.FuncCallExpr) ([]ir.Element, error) {
	args := make([]ir.Element, len(expr.Args))
	for i, arg := range expr.Args {
		el, err := fitp.EvalExpr(arg)
		if err != nil {
			return nil, err
		}
		args[i] = el
	}
	return fn.Call(fitp.env, expr, args)
}

func set(fitp *FileScope, tok token.Token, dest ir.Storage, value ir.Element) error {
	switch destT := dest.(type) {
	case *ir.LocalVarStorage:
		if !ir.ValidIdent(destT.Src) {
			return nil
		}
		if tok == token.ILLEGAL {
			return nil
		}
		if tok == token.DEFINE {
			fitp.ctx.CurrentFrame().Define(destT.Src.Name, value)
			return nil
		}
		return fitp.ctx.CurrentFrame().Assign(destT.Src.Name, value)
	case *ir.StructFieldStorage:
		receiver, err := evalExpr(fitp, destT.Sel.X)
		if err != nil {
			return err
		}
		strt, ok := elements.Underlying(receiver).(*elements.Struct)
		if !ok {
			return fmterr.Errorf(fitp.File().FileSet(), dest.Node(), "cannot convert %T to %T", receiver, strt)
		}
		strt.SetField(destT.Sel.Src.Sel.Name, value)
		return nil
	case *ir.FieldStorage:
		return fitp.ctx.CurrentFrame().Assign(destT.Field.Name.Name, value)
	case *ir.AssignExpr:
		return fitp.ctx.CurrentFrame().Assign(destT.NameDef().Name, value)
	case *ir.AssignCallResult:
		return fitp.ctx.CurrentFrame().Assign(destT.NameDef().Name, value)
	default:
		return fmterr.Errorf(fitp.File().FileSet(), dest.Node(), "cannot assign %v to %T: not supported", value, destT)
	}
}

// ToSingleElement packs multiple elements into a tuple.
// If the slice els contains only one element, this element is returned.
func ToSingleElement(ctx ir.Evaluator, node ir.Node, els []ir.Element) (ir.Element, error) {
	switch len(els) {
	case 0:
		return nil, nil
	case 1:
		return els[0], nil
	default:
		return NewTuple(els), nil
	}

}

func dimsAsElements(fitp *FileScope, expr ir.AssignableExpr, dims []int) ([]evaluator.NumericalElement, error) {
	els := make([]evaluator.NumericalElement, len(dims))
	for i, di := range dims {
		val, err := values.AtomIntegerValue[int64](ir.IntLenType(), int64(di))
		if err != nil {
			return nil, err
		}
		els[i], err = fitp.Evaluator().ElementFromAtom(fitp.File(), expr, val)
		if err != nil {
			return nil, err
		}
	}
	return els, nil
}

func rankOf(env evaluator.Env, src ir.Node, typ ir.ArrayType) (ir.ArrayRank, error) {
	switch rank := typ.Rank().(type) {
	case *ir.Rank:
		return rank, nil
	case *ir.RankInfer:
		if rank.Rnk == nil {
			return nil, fmterr.Errorf(env.ExprEval().File().FileSet(), src.Node(), "array rank has not been resolved")
		}
		return rank.Rnk, nil
	default:
		return nil, fmterr.Errorf(env.ExprEval().File().FileSet(), src.Node(), "rank %T not supported", rank)
	}
}
