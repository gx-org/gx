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

package grapheval

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/internal/togo"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/src/gx/ast"
)

type (
	argFetcher interface {
		LastValue(*togo.FuncInputs) (any, error)
	}

	parameterFetcher struct {
		paramIndex int
	}

	receiverFetcher struct{}
)

func (f parameterFetcher) LastValue(in *togo.FuncInputs) (any, error) {
	return in.Args[f.paramIndex], nil
}

func (f receiverFetcher) LastValue(in *togo.FuncInputs) (any, error) {
	return in.Receiver, nil
}

type inputVisitor struct {
	ao   *Ops
	file *ir.File
}

func newInputVisitor(ev *Ops, file *ir.File) *inputVisitor {
	return &inputVisitor{ao: ev, file: file}
}

// ArgGX represents a GX argument.
func (vis *inputVisitor) visitArg(field *ir.Field, index int, proxy hostio.Value) (ir.Element, error) {
	return vis.newRootArg(field, parameterFetcher{paramIndex: index}, proxy)
}

// Receiver represents a GX function call receiver.
func (vis *inputVisitor) visitReceiver(field *ir.Field, proxy ir.Element) (ir.Element, error) {
	return vis.newRootArg(field, receiverFetcher{}, proxy)
}

type (
	parentArgument interface {
		togo.RuntimeValue
		Name() string
		Ops() *Ops
	}

	// rootArgument is an argument passed to the GX interpreter from the host language.
	rootArgument struct {
		argFetcher
		vis  *inputVisitor
		name string
		typ  ir.Type
	}
)

var _ parentArgument = (*rootArgument)(nil)

func (vis *inputVisitor) newRootArg(field *ir.Field, fetcher argFetcher, proxy ir.Element) (ir.Element, error) {
	n := &rootArgument{
		vis:        vis,
		argFetcher: fetcher,
		name:       field.Name.Name,
		typ:        field.Type(),
	}
	return vis.newArg(n, field.Type(), proxy)
}

func (a *rootArgument) Name() string {
	return a.name
}

func (a *rootArgument) Type() ir.Type {
	return a.typ
}

func (a *rootArgument) Ops() *Ops {
	return a.vis.ao
}

func (vis *inputVisitor) newArg(parent parentArgument, typ ir.Type, proxy ir.Element) (ir.Element, error) {
	if proxy.Type().Kind() == irkind.Unknown {
		return proxy, nil
	}
	switch typT := typ.(type) {
	case *ir.NamedType:
		return vis.newNamedTypeArgument(parent, typT, proxy)
	case *ir.StructType:
		return vis.newStructArgument(parent, typT, proxy)
	case *ir.SliceType:
		return vis.newSliceArgument(parent, typT, proxy)
	case ir.ArrayType:
		withShape, err := cast.To[processor.Shaper](proxy)
		if err != nil {
			return nil, err
		}
		return vis.newArrayArgument(parent, typT, withShape.Shape())
	default:
		return nil, errors.Errorf("argument type %T not supported", typT)
	}
}

type namedTypeArgument struct {
	parent parentArgument
	typ    *ir.NamedType
}

func (vis *inputVisitor) newNamedTypeArgument(parent parentArgument, typ *ir.NamedType, el ir.Element) (*elements.NamedType, error) {
	arg := &namedTypeArgument{
		parent: parent,
		typ:    typ,
	}
	named, ok := el.(engine.NamedType)
	if !ok {
		return nil, errors.Errorf("element %T is not a named type element", el)
	}
	under, err := named.Under()
	if err != nil {
		return nil, err
	}
	recv, err := vis.newArg(arg, typ.Underlying.Val(), under)
	if err != nil {
		return nil, err
	}
	return elements.NewNamedType(arg.typ, recv), nil
}

func (arg *namedTypeArgument) Name() string {
	return arg.parent.Name() + "." + arg.typ.Name()
}

func (arg *namedTypeArgument) Type() ir.Type {
	return arg.typ
}

func (arg *namedTypeArgument) LastValue(ctx *togo.FuncInputs) (any, error) {
	return arg.parent.LastValue(ctx)
}

func (arg *namedTypeArgument) Ops() *Ops {
	return arg.parent.Ops()
}

type (
	structArgument struct {
		parentArgument
		typ   *ir.StructType
		proxy engine.Selector
	}

	fieldSelectorArgument struct {
		parent    *structArgument
		fieldName string
		field     *ir.Field
	}
)

var _ parentArgument = (*fieldSelectorArgument)(nil)

func (vis *inputVisitor) newStructArgument(parent parentArgument, typ *ir.StructType, proxy ir.Element) (*elements.Struct, error) {
	sel, err := cast.To[engine.Selector](proxy)
	if err != nil {
		return nil, errors.Errorf("%T does not support %s", proxy, reflect.TypeFor[engine.Selector]().Name())
	}
	structArg := &structArgument{
		parentArgument: parent,
		typ:            typ,
		proxy:          sel,
	}
	fields := make(map[string]ir.Element, typ.NumFields())
	for _, field := range typ.Fields.Fields() {
		name := field.Name.Name
		selector := &fieldSelectorArgument{
			parent:    structArg,
			fieldName: name,
			field:     field,
		}
		fieldExpr := &ir.SelectorExpr{
			Src: &ast.SelectorExpr{
				Sel: &ast.Ident{Name: name},
			},
			Stor: &ir.FieldStorage{Field: field},
		}
		fieldProxy, err := sel.Select(nil, fieldExpr)
		if err != nil {
			return nil, err
		}
		if fieldProxy == nil {
			return nil, errors.Errorf("field %s has no value", name)
		}
		field, err := vis.newArg(selector, field.Type(), fieldProxy)
		if err != nil {
			return nil, err
		}
		fields[name] = field
	}
	return elements.NewStruct(typ, fields), nil
}

func (sel *fieldSelectorArgument) Name() string {
	return sel.parent.Name() + "." + sel.fieldName
}

func (sel *fieldSelectorArgument) LastValue(ctx *togo.FuncInputs) (any, error) {
	structValue, err := togo.RtValueT[engine.Selector](ctx, sel.parent)
	if err != nil {
		return nil, err
	}
	fieldVal, err := structValue.Select(nil, &ir.SelectorExpr{
		Src: &ast.SelectorExpr{Sel: &ast.Ident{Name: sel.fieldName}},
	})
	if err != nil {
		return nil, err
	}
	if fieldVal == nil {
		return nil, errors.Errorf("no field %s in structure instance passed as argument %s (type %s)", sel.fieldName, sel.parent.Name(), sel.parent.typ.ReferString(nil))
	}
	return fieldVal, nil
}

func (sel *fieldSelectorArgument) Ops() *Ops {
	return sel.parent.Ops()
}

func (sel *fieldSelectorArgument) Type() ir.Type {
	return sel.field.Type()
}

type (
	sliceArgument struct {
		parentArgument
		typ   *ir.SliceType
		proxy ir.FixedSlice
	}

	indexSelectorArgument struct {
		parent *sliceArgument
		index  int
		proxy  ir.Element
	}
)

func (vis *inputVisitor) newSliceArgument(parent parentArgument, typ *ir.SliceType, proxy ir.Element) (*elements.Slice, error) {
	fixed, ok := proxy.(ir.FixedSlice)
	if !ok {
		return nil, errors.Errorf("%T does not support %s", proxy, reflect.TypeFor[ir.FixedSlice]().Name())
	}
	sliceArg := &sliceArgument{
		parentArgument: parent,
		typ:            typ,
		proxy:          fixed,
	}
	elType, ok := typ.ElementType()
	if !ok {
		return nil, errors.Errorf("atomic type %s cannot be indexed", typ.ReferString(nil))
	}
	args := make([]ir.Element, fixed.Len())
	for i, iProxy := range fixed.Elements() {
		idxSel := &indexSelectorArgument{
			parent: sliceArg,
			index:  i,
			proxy:  iProxy,
		}
		var err error
		args[i], err = vis.newArg(idxSel, elType, iProxy)
		if err != nil {
			return nil, err
		}

	}
	return elements.NewSlice(typ, args)
}

func (sel *indexSelectorArgument) Type() ir.Type {
	tp, _ := sel.parent.typ.ElementType()
	return tp
}

func (sel *indexSelectorArgument) Name() string {
	return sel.parent.Name() + "." + fmt.Sprintf("[%d]", sel.index)
}

func (sel *indexSelectorArgument) LastValue(ctx *togo.FuncInputs) (any, error) {
	sliceValue, err := togo.RtValueT[*values.Slice](ctx, sel.parent)
	if err != nil {
		return nil, err
	}
	val := sliceValue.Element(sel.index)
	if val == nil {
		return nil, errors.Errorf("no element at %d in slice instance passed as argument %s (type %s)", sel.index, sel.parent.Name(), sel.parent.typ.ReferString(nil))
	}
	return val, nil
}

func (sel *indexSelectorArgument) Ops() *Ops {
	return sel.parent.Ops()
}

var _ parentArgument = (*indexSelectorArgument)(nil)

type arrayArgument struct {
	parentArgument
	typ ir.ArrayType

	file  *ir.File
	shape shapes.Shape
	index int
	node  *BackendNode
}

var (
	_ processor.Argument              = (*arrayArgument)(nil)
	_ engine.NumericalElement         = (*arrayArgument)(nil)
	_ materialise.ElementMaterialiser = (*arrayArgument)(nil)
	_ flatten.Unflattener             = (*arrayArgument)(nil)
	_ elements.Slicer                 = (*arrayArgument)(nil)
	_ ir.WithLength                   = (*arrayArgument)(nil)
	_ elements.WithAxes               = (*arrayArgument)(nil)
	_ elements.EvalShaper             = (*arrayArgument)(nil)
	_ graphNode                       = (*arrayArgument)(nil)
	_ togo.RuntimeValue               = (*arrayArgument)(nil)
)

// NewArrayArgument creates a new argument element that the graph can also use as an argument.
func (ao *Ops) NewArrayArgument(file *ir.File, parent parentArgument, typ ir.ArrayType, sh shapes.Shape) (engine.NumericalElement, error) {
	vis := &inputVisitor{file: file, ao: ao}
	return vis.newArrayArgument(parent, typ, sh)
}

func (n *arrayArgument) out() compute.Value {
	return n.node.out()
}

// NewArrayArgument creates a new argument element that the graph can also use as an argument.
func (vis *inputVisitor) newArrayArgument(parent parentArgument, typ ir.ArrayType, sh shapes.Shape) (engine.NumericalElement, error) {
	n := &arrayArgument{
		parentArgument: parent,
		typ:            typ,
		shape:          sh,
		file:           vis.file,
	}
	n.index = vis.ao.process.RegisterArg(n)
	return n, nil
}

func (n *arrayArgument) Name() string {
	return fmt.Sprintf("%s.%d", n.parentArgument.Name(), n.index)
}

// UnaryOp applies a unary operator on x.
func (n *arrayArgument) UnaryOp(env *engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	node, err := n.materialise(n.parentArgument.Ops())
	if err != nil {
		return nil, err
	}
	return node.UnaryOp(env, expr)
}

// BinaryOp applies a binary operator to x and y.
// Note that the receiver can be either the left or right argument.
func (n *arrayArgument) BinaryOp(env *engine.Env, expr *ir.BinaryExpr, y engine.NumericalElement) (engine.NumericalElement, error) {
	node, err := n.materialise(n.parentArgument.Ops())
	if err != nil {
		return nil, err
	}
	return node.BinaryOp(env, expr, y)
}

// Cast an element into a given data type.
func (n *arrayArgument) Cast(env *engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	node, err := n.materialise(n.parentArgument.Ops())
	if err != nil {
		return nil, err
	}
	return node.Cast(env, expr, target)
}

// Reshape an element.
func (n *arrayArgument) Reshape(env *engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	node, err := n.materialise(n.parentArgument.Ops())
	if err != nil {
		return nil, err
	}
	return node.Reshape(env, expr, axisLengths)
}

func (n *arrayArgument) materialise(mat materialise.Materialiser) (*BackendNode, error) {
	if n.node != nil {
		return n.node, nil
	}
	op, err := mat.Graph().Parameter(n.Name(), n.shape, nil)
	if err != nil {
		return nil, err
	}
	n.node, err = NewBackendNode(mat.(*Ops), n.typ, op)
	if err != nil {
		return nil, err
	}
	return n.node, nil
}

// Materialise returns the element with all its values from the graph.
func (n *arrayArgument) Materialise(mat materialise.Materialiser) (materialise.Node, error) {
	return n.materialise(mat)
}

// Shape returns the shape of the element.
func (n *arrayArgument) Shape() shapes.Shape {
	return n.shape
}

// Shape returns the shape of the element.
func (n *arrayArgument) EvalShape() (shapes.Shape, error) {
	return n.shape, nil
}

func (n *arrayArgument) Axes(ev ir.Evaluator) (*elements.Slice, error) {
	return n.node.ao.axesFromShape(ev, n.shape.Dimensions)
}

func (n *arrayArgument) Length(ev ir.Evaluator) (int, error) {
	return shapeLength(n.Shape())
}

func (n *arrayArgument) ToDeviceHandle(plat compute.Backend, dev compute.DeviceNum, in *togo.FuncInputs) (compute.Buffer, error) {
	array, err := togo.RtValueT[hostio.Array](in, n)
	if err != nil {
		return nil, err
	}
	return toDevice(plat, dev, array)
}

func toDevice(plat compute.Backend, dev compute.DeviceNum, arr hostio.Array) (compute.Buffer, error) {
	deviceArray, err := arr.ToDevice(plat, dev)
	if err != nil {
		return nil, err
	}
	return deviceArray.Buffer(), nil
}

// NumericalConstant returns the value of a constant represented by a node.
func (n *arrayArgument) LastValue(in *togo.FuncInputs) (any, error) {
	return n.parentArgument.LastValue(in)
}

func (n *arrayArgument) Copy() engine.Copier {
	return n
}

func (n *arrayArgument) Type() ir.Type {
	return n.typ
}

// Unflatten consumes the next handles to return a GX value.
func (n *arrayArgument) Unflatten(handles *flatten.Parser) (hostio.Value, error) {
	return handles.ParseArray(n.typ)
}

// SliceAt of the value on the first axis given an index.
func (n *arrayArgument) SliceAt(env *engine.Env, expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
	node, err := n.materialise(n.Ops())
	if err != nil {
		return nil, err
	}
	return node.SliceAt(env, expr, index)
}

// Slice the argument.
func (n *arrayArgument) Slice(env *engine.Env, expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	return nil, errors.Errorf("not implemented for %T", n)
}

// SliceArray of the value on the first axis given an index.
func (n *arrayArgument) SliceArray(expr ir.Expr, index engine.NumericalElement) (engine.NumericalElement, error) {
	node, err := n.materialise(n.Ops())
	if err != nil {
		return nil, err
	}
	return node.SliceArray(expr, index)
}
