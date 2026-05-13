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
	"fmt"
	"go/token"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp/numbers"
)

func evalBlockStmt(ctx *Interpreter, body *ir.BlockStmt) ([]ir.Element, bool, error) {
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

func evalStmt(fitp *Interpreter, node ir.Stmt) ([]ir.Element, bool, error) {
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

func evalRangeForLoopOverInteger[T dtype.AlgebraType](fitp *Interpreter, stmt *ir.RangeStmt, toValue valuer) ([]ir.Element, bool, error) {
	toValueT := toValue.(valuerT[T])
	indexType := ir.TypeFromKind(toValueT.kind)
	val, err := evalAtom[T](fitp, stmt.X)
	if err != nil {
		return nil, true, err
	}
	fitp.Context().PushBlockFrame()
	defer fitp.Context().PopFrame()
	for i := T(0); i < val; i++ {
		iExpr := &ir.AtomicValueT[T]{
			Src: stmt.X.Expr(),
			Val: i,
			Typ: indexType,
		}
		iValue, err := toValueT.toAtomValue(iExpr.Type(), i)
		if err != nil {
			return nil, true, err
		}
		iElement, err := fitp.elementFromAtom(iExpr, iValue)
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

func evalRangeStmtInteger(fitp *Interpreter, stmt *ir.RangeStmt, xKind irkind.Kind) ([]ir.Element, bool, error) {
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

func evalRangeStmtForLoopOverArray[T dtype.AlgebraType](fitp *Interpreter, stmt *ir.RangeStmt, toValue valuer) ([]ir.Element, bool, error) {
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
		return nil, false, fmterr.Error(fitp.File().FileSet(), stmt.Node(), err)
	}
	for i := 0; i < arrayShape.AxisLengths[0]; i++ {
		iExpr := &ir.AtomicValueT[T]{
			Src: stmt.X.Expr(),
			Val: T(i),
			Typ: indexType,
		}
		iValue, err := toValueT.toAtomValue(iExpr.Type(), T(i))
		if err != nil {
			return nil, false, err
		}
		iElement, err := fitp.elementFromAtom(iExpr, iValue)
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

func evalRangeStmtArray(fitp *Interpreter, stmt *ir.RangeStmt) ([]ir.Element, bool, error) {
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

func evalRangeStmt(fitp *Interpreter, stmt *ir.RangeStmt) ([]ir.Element, bool, error) {
	kind := stmt.X.Type().Kind()
	if irkind.IsRangeOk(kind) {
		return evalRangeStmtInteger(fitp, stmt, kind)
	}
	if kind == irkind.Array {
		return evalRangeStmtArray(fitp, stmt)
	}
	return nil, true, errors.Errorf("cannot range over %s", kind.String())
}

func evalIfStmt(fitp *Interpreter, stmt *ir.IfStmt) ([]ir.Element, bool, error) {
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

func evalAssignExprStmt(fitp *Interpreter, stmt *ir.AssignExprStmt) error {
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

func evalAssignCallStmt(fitp *Interpreter, stmt *ir.AssignCallStmt) error {
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

func evalReturnStmt(fitp *Interpreter, ret *ir.ReturnStmt) ([]ir.Element, bool, error) {
	if len(ret.Results) == 0 {
		// Naked return.
		fields := fitp.Context().CurrentFunc().FuncType().Results.Fields()
		nodes := make([]ir.Element, len(fields))
		fr := fitp.Context().CurrentFrame()
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

func evalCastToScalarExpr(fitp *Interpreter, expr ir.TypeCastExpr, x engine.NumericalElement, targetType ir.ArrayType) (ir.Element, error) {
	xShape, err := elements.ShapeFromElement(x)
	if err != nil {
		return nil, fmterr.Error(fitp.File().FileSet(), expr.Node(), err)
	}
	if len(xShape.AxisLengths) > 0 {
		return x.Reshape(fitp.env, expr, nil)
	}
	return x.Cast(fitp.env, expr, targetType)
}

func evalArrayAxis(fitp *Interpreter, src ir.Node, axLen ir.AxisLengths) ([]engine.NumericalElement, error) {
	el, err := evalExpr(fitp, axLen.AsExpr())
	if err != nil {
		return nil, err
	}
	switch elT := elements.Underlying(el).(type) {
	case engine.NumericalElement:
		return []engine.NumericalElement{elT}, nil
	case *elements.Slice:
		numEls := make([]engine.NumericalElement, elT.Len())
		for i, eli := range elT.Elements() {
			var err error
			numEls[i], err = elements.ToNumericalElement(eli)
			if err != nil {
				return nil, err
			}
		}
		return numEls, nil
	default:
		return nil, errors.Errorf("cannot %T to an axis length: not supported", elT)
	}
}

func evalArrayAxes(fitp *Interpreter, src ir.Node, typ ir.ArrayType) ([]engine.NumericalElement, error) {
	rank, err := rankOf(fitp.env, src, typ)
	if err != nil {
		return nil, err
	}
	var axes []engine.NumericalElement
	for _, axLen := range rank.Axes() {
		axis, err := evalArrayAxis(fitp, src, axLen)
		if err != nil {
			return nil, err
		}
		axes = append(axes, axis...)
	}
	return axes, nil
}

var one, _ = values.AtomIntegerValue(ir.IntLenType(), ir.Int(1))

func evalCastAtomToArrayExpr(fitp *Interpreter, expr ir.TypeCastExpr, x engine.NumericalElement, axes []engine.NumericalElement) (ir.Element, error) {
	srcExpr := elements.NewExprAt(fitp.File(), expr)
	arrayOps := fitp.Engine().ArrayOps()
	shapeOfOnes := make([]engine.NumericalElement, len(axes))
	for i := range axes {
		var err error
		shapeOfOnes[i], err = fitp.elementFromAtom(expr, one)
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

func evalCastToArrayExpr(fitp *Interpreter, expr ir.TypeCastExpr, x engine.NumericalElement, targetType ir.ArrayType) (ir.Element, error) {
	origType := expr.Orig().Type()
	origRank, xDType := ir.Shape(origType)
	targetDType, err := toConcreteType(fitp.Context(), expr.Node(), fitp.Context().CurrentFrame(), targetType.DataType())
	if err != nil {
		return nil, err
	}
	targetArrayType := ir.NewArrayType(expr.Expr(), targetDType, targetType.Rank())
	expr = &ir.CastExpr{
		Src: expr.Expr(),
		Typ: targetArrayType,
		X:   expr.Orig(),
	}
	targetDTypeKind := targetDType.Kind()
	if xDType.Kind() != targetDTypeKind {
		var err error
		x, err = x.Cast(fitp.env, expr, targetDType)
		if err != nil {
			return nil, fmterr.Errorf(fitp.File().FileSet(), expr.Node(), "cannot cast expression %q to %s: %v", expr.SourceString(fitp.File()), targetDType, err)
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
		return nil, fmterr.Error(fitp.File().FileSet(), expr.Node(), err)
	}
	sourceType, ok := origType.(ir.ArrayType)
	if !ok {
		return nil, fmterr.Errorf(fitp.File().FileSet(), expr.Node(), "cannot cast %T to %s", origType, reflect.TypeFor[ir.ArrayType]().Name())
	}
	if sourceType.DataType().Kind() == targetDTypeKind {
		return reshape, nil
	}
	return reshape.Cast(fitp.env, expr, targetDType)
}

func evalCastExpr(fitp *Interpreter, expr ir.TypeCastExpr) (ir.Element, error) {
	x, err := evalExpr(fitp, expr.Orig())
	if err != nil {
		return nil, err
	}
	target := expr.Type()
	if named, ok := target.(*ir.NamedType); ok {
		return fun.NewNamedType(fitp.NewFunc, named, x), nil
	}
	arrayType, ok := target.(ir.ArrayType)
	if !ok {
		return nil, fmterr.Errorf(fitp.File().FileSet(), expr.Node(), "cast to %s not supported", target.ReferString(fitp.File()))
	}
	xNum, ok := elements.Underlying(x).(engine.NumericalElement)
	if !ok {
		return nil, fmterr.Errorf(fitp.File().FileSet(), expr.Node(), "cannot cast element of type %T to %s", x, reflect.TypeFor[engine.NumericalElement]().Name())
	}
	if arrayType.Rank().IsAtomic() {
		return evalCastToScalarExpr(fitp, expr, xNum, arrayType)
	}
	return evalCastToArrayExpr(fitp, expr, xNum, arrayType)
}

func evalUnaryExpression(fitp *Interpreter, expr *ir.UnaryExpr) (ir.Element, error) {
	x, err := evalNumExpr(fitp, expr.X)
	if err != nil {
		return nil, err
	}
	if ir.IsStatic(expr.Type()) {
		return nil, errors.Errorf("not supported")
	}
	return x.UnaryOp(fitp.env, expr)
}

func evalBinaryExpression(fitp *Interpreter, expr *ir.BinaryExpr) (ir.Element, error) {
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

func evalStructLiteral(fitp *Interpreter, expr *ir.StructLitExpr) (ir.Element, error) {
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

func evalSliceExpr(fitp *Interpreter, expr *ir.SliceExpr) (ir.Element, error) {
	el, err := evalExpr(fitp, expr.X)
	if err != nil {
		return nil, err
	}
	var low, high engine.NumericalElement
	if expr.Low != nil {
		low, err = evalNumExpr(fitp, expr.Low)
		if err != nil {
			return nil, err
		}
	}
	if expr.High != nil {
		high, err = evalNumExpr(fitp, expr.High)
		if err != nil {
			return nil, err
		}
	}
	slice, isSlice := el.(*elements.Slice)
	if !isSlice {
		return nil, fmterr.Errorf(fitp.File().FileSet(), expr.Node(), "index expression not supported for %s expression", expr.X.SourceString(nil))
	}
	return slice.Slice(expr, low, high)
}

func evalSliceLiteral(fitp *Interpreter, expr *ir.SliceLitExpr) (ir.Element, error) {
	els := make([]ir.Element, len(expr.Elts))
	for i, expr := range expr.Elts {
		elt, err := evalExpr(fitp, expr)
		if err != nil {
			return nil, err
		}
		els[i] = elt
	}
	return elements.NewSlice(expr.Type(), els)
}

// evalExpr evaluates an expression within the Context.
func evalExpr(fitp *Interpreter, expr ir.Expr) (_ ir.Element, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot evaluate expression %q:\n%w", expr.SourceString(fitp.File()), err)
		}
	}()
	if expr == nil {
		return nil, errors.Errorf("cannot evaluate a nil expression")
	}
	switch exprT := expr.(type) {
	case *ir.ArrayLitExpr:
		return evalArrayLiteral(fitp, exprT)
	case *ir.AxisExpr:
		return evalExpr(fitp, exprT.X)
	case *ir.AxisInfer:
		return evalExpr(fitp, exprT.X.AsExpr())
	case *ir.NilCastExpr:
		return elements.NewNil(exprT.Typ), nil
	case *ir.NumberCastExpr:
		return evalNumberCastExpr(fitp, exprT)
	case *ir.SliceLitExpr:
		return evalSliceLiteral(fitp, exprT)
	case *ir.SliceExpr:
		return evalSliceExpr(fitp, exprT)
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
	case *ir.UnpackExpr:
		return evalExpr(fitp, exprT.X)
	case *ir.ParenExpr:
		return evalExpr(fitp, exprT.X)
	case *ir.BinaryExpr:
		return evalBinaryExpression(fitp, exprT)
	case *ir.Ident:
		return evalIdent(fitp, exprT)
	case *ir.SelectorExpr:
		return evalSelectorExpr(fitp, exprT)
	case *ir.FuncLit:
		return fitp.env.FuncEval().NewFuncLit(exprT, fitp.Context()), nil
	case *ir.MacroCallExpr:
		return fitp.env.FuncEval().NewFunc(exprT.F, nil), nil
	case *ir.IndexExpr:
		return evalIndexExpr(fitp, exprT)
	case *ir.EinsumExpr:
		return evalEinsumExpr(fitp, exprT)
	case *ir.StringLiteral:
		return elements.NewStringFromLit(exprT)
	case *ir.NumberFloat:
		return numbers.NewFloat(fitp.env, exprT, exprT.Val)
	case *ir.NumberInt:
		return numbers.NewInt(fitp.env, exprT, exprT.Val)
	case ir.AtomicValue:
		return evalAtomicValue(fitp, exprT)
	case *ir.PackageRef:
		return fitp.Context().CurrentFrame().Find(exprT.X.Src)
	case *ir.FuncValExpr:
		return evalFuncValExpr(fitp, exprT)
	case *ir.TypeValExpr:
		return exprT, nil
	default:
		return nil, fmterr.Errorf(fitp.File().FileSet(), expr.Node(), "cannot evaluate GX expression: %T not supported", expr)
	}
}

func evalFuncValExpr(fitp *Interpreter, expr *ir.FuncValExpr) (ir.Element, error) {
	lit, isLit := expr.Func().(*ir.FuncLit)
	if isLit {
		return fitp.env.FuncEval().NewFuncLit(lit, fitp.Context()), nil
	}
	return fitp.NewFunc(expr.Func(), nil), nil
}

func evalIdent(fitp *Interpreter, ref *ir.Ident) (ir.Element, error) {
	if ref.Src == nil {
		return ref.Stor, nil
	}
	return fitp.Context().CurrentFrame().Find(ref.Src)
}

func evalNumExpr(fitp *Interpreter, expr ir.Expr) (engine.NumericalElement, error) {
	el, err := evalExpr(fitp, expr)
	if err != nil {
		return nil, err
	}
	return elements.ToNumericalElement(el)
}

func evalNumberCastExpr(fitp *Interpreter, expr *ir.NumberCastExpr) (engine.NumericalElement, error) {
	number, err := evalNumExpr(fitp, expr.X)
	if err != nil {
		return nil, err
	}
	tp := expr.Typ
	if tpParam, ok := expr.Typ.(*ir.TypeParam); ok {
		tpEl, err := fitp.Context().CurrentFrame().Find(tpParam.Field.Name)
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

func evalSelectorExpr(fitp *Interpreter, ref *ir.SelectorExpr) (ir.Element, error) {
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

func evalIndexExpr(fitp *Interpreter, ref *ir.IndexExpr) (ir.Element, error) {
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
	return slicer.SliceAt(ref, index)
}

func evalEinsumExpr(fitp *Interpreter, ref *ir.EinsumExpr) (ir.Element, error) {
	x, err := evalNumExpr(fitp, ref.X)
	if err != nil {
		return nil, err
	}
	y, err := evalNumExpr(fitp, ref.Y)
	if err != nil {
		return nil, err
	}
	return fitp.Engine().ArrayOps().Einsum(fitp, ref, x, y)
}

func evalAtom[T dtype.GoDataType](fitp *Interpreter, expr ir.Expr) (val T, err error) {
	el, err := evalExpr(fitp, expr)
	if err != nil {
		var zero T
		return zero, err
	}
	return elements.ConstantScalarFromElement[T](el)
}

func evalCallee(fitp *Interpreter, callee ir.Callee) (fun.Func, error) {
	switch calleeT := callee.(type) {
	case *ir.FuncValExpr:
		fnNode, err := fitp.EvalExpr(calleeT.X())
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

func evalCallExpr(fitp *Interpreter, expr *ir.FuncCallExpr) (ir.Element, error) {
	outs, err := evalCall(fitp, expr)
	if err != nil {
		return nil, err
	}
	return ToSingleElement(fitp, expr, outs)
}

func evalArgExpr(fitp *Interpreter, expr ir.Expr) ([]ir.Element, error) {
	el, err := evalExpr(fitp, expr)
	if err != nil {
		return nil, err
	}
	if _, isUnpack := expr.(*ir.UnpackExpr); !isUnpack {
		return []ir.Element{el}, err
	}
	sl, err := elements.SliceFromElement(el)
	if err != nil {
		return nil, err
	}
	return sl.Elements(), nil
}

func evalCall(fitp *Interpreter, expr *ir.FuncCallExpr) ([]ir.Element, error) {
	callee, err := evalCallee(fitp, expr.Callee)
	if err != nil {
		return nil, err
	}
	// Evaluate the arguments to pass to the function.
	var args []ir.Element
	for _, arg := range expr.Args {
		el, err := evalArgExpr(fitp, arg)
		if err != nil {
			return nil, err
		}
		args = append(args, el...)
	}
	return callee.Call(fitp.env, expr, args)
}

func evalFuncCall(fitp *Interpreter, fn fun.Func, expr *ir.FuncCallExpr) ([]ir.Element, error) {
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

func set(fitp *Interpreter, tok token.Token, dest ir.Storage, value ir.Element) error {
	ctx := fitp.Context()
	switch destT := dest.(type) {
	case *ir.LocalVarStorage:
		if !ir.ValidIdent(destT.Src) {
			return nil
		}
		if tok == token.ILLEGAL {
			return nil
		}
		if tok == token.DEFINE {
			ctx.CurrentFrame().Define(destT.Src, value)
			return nil
		}
		return ctx.CurrentFrame().Assign(destT.Src.Name, value)
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
		return ctx.CurrentFrame().Assign(destT.Field.Name.Name, value)
	case *ir.AssignExpr:
		return ctx.CurrentFrame().Assign(destT.NameDef().Name, value)
	case *ir.AssignCallResult:
		return ctx.CurrentFrame().Assign(destT.NameDef().Name, value)
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

func dimsAsElements(fitp *Interpreter, expr ir.Expr, dims []int) ([]engine.NumericalElement, error) {
	els := make([]engine.NumericalElement, len(dims))
	for i, di := range dims {
		val, err := values.AtomIntegerValue[int64](ir.IntLenType(), int64(di))
		if err != nil {
			return nil, err
		}
		els[i], err = fitp.elementFromAtom(expr, val)
		if err != nil {
			return nil, err
		}
	}
	return els, nil
}

func rankOf(env engine.Env, src ir.Node, typ ir.ArrayType) (ir.ArrayRank, error) {
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
