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
	"go/ast"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/interp/materialise"
)

type (
	argFetcher interface {
		ValueFromContext(*values.FuncInputs) (ir.Element, error)
	}

	parameterFetcher struct {
		paramIndex int
	}

	receiverFetcher struct{}
)

func (f parameterFetcher) ValueFromContext(in *values.FuncInputs) (ir.Element, error) {
	return in.Args[f.paramIndex], nil
}

func (f receiverFetcher) ValueFromContext(in *values.FuncInputs) (ir.Element, error) {
	return in.Receiver, nil
}

type inputVisitor struct {
	proc *processor.Processor
	file *ir.File
}

func newInputVisitor(file *ir.File, proc *processor.Processor) *inputVisitor {
	return &inputVisitor{file: file, proc: proc}
}

// ArgGX represents a GX argument.
func (vis *inputVisitor) visitArg(field *ir.Field, index int, proxy ir.Element) (ir.Element, error) {
	return vis.newRootArg(field, parameterFetcher{paramIndex: index}, proxy)
}

// Receiver represents a GX function call receiver.
func (vis *inputVisitor) visitReceiver(field *ir.Field, proxy ir.Element) (ir.Element, error) {
	return vis.newRootArg(field, receiverFetcher{}, proxy)
}

type (
	parentArgument interface {
		Name() string
		ValueFromContext(*values.FuncInputs) (ir.Element, error)
	}

	// rootArgument is an argument passed to the GX interpreter from the host language.
	rootArgument struct {
		argFetcher
		name string
		typ  ir.Type
	}
)

var _ parentArgument = (*rootArgument)(nil)

func (vis *inputVisitor) newRootArg(field *ir.Field, fetcher argFetcher, proxy ir.Element) (ir.Element, error) {
	n := &rootArgument{
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

func (vis *inputVisitor) newArg(parent parentArgument, typ ir.Type, proxy ir.Element) (ir.Element, error) {
	switch typT := typ.(type) {
	case *ir.NamedType:
		return vis.newNamedTypeArgument(parent, typT, proxy)
	case *ir.StructType:
		return vis.newStructArgument(parent, typT, proxy)
	case *ir.SliceType:
		return vis.newSliceArgument(parent, typT, proxy)
	case ir.ArrayType:
		return vis.newArrayArgument(parent, typT, proxy)
	default:
		return nil, errors.Errorf("argument type %T not supported", typT)
	}
}

type namedTypeArgument struct {
	parent parentArgument
	typ    *ir.NamedType
}

func (vis *inputVisitor) newNamedTypeArgument(parent parentArgument, typ *ir.NamedType, el ir.Element) (*interp.NamedType, error) {
	arg := &namedTypeArgument{
		parent: parent,
		typ:    typ,
	}
	named, ok := el.(interp.NType)
	if !ok {
		return nil, errors.Errorf("element %T is not a named type element", el)
	}
	recv, err := vis.newArg(arg, typ.Underlying.Typ, named.Under())
	if err != nil {
		return nil, err
	}
	recvCopier, ok := recv.(interp.Copier)
	if !ok {
		return nil, errors.Errorf("element %T cannot be used as a receiver", recv)
	}
	return interp.NewNamedType(interp.NewRunFunc, arg.typ, recvCopier), nil
}

func (arg *namedTypeArgument) Name() string {
	return arg.parent.Name() + "." + arg.typ.Name()
}

func (arg *namedTypeArgument) ValueFromContext(ctx *values.FuncInputs) (ir.Element, error) {
	return arg.parent.ValueFromContext(ctx)
}

type (
	structArgument struct {
		parentArgument
		typ   *ir.StructType
		proxy interp.Selector
	}

	fieldSelectorArgument struct {
		parent    *structArgument
		fieldName string
		field     *ir.Field
	}
)

var _ parentArgument = (*fieldSelectorArgument)(nil)

func (vis *inputVisitor) newStructArgument(parent parentArgument, typ *ir.StructType, proxy ir.Element) (*interp.Struct, error) {
	sel, ok := proxy.(interp.Selector)
	if !ok {
		return nil, errors.Errorf("%T does not support %s", proxy, reflect.TypeFor[interp.Selector]().Name())
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
		fieldProxy, err := sel.Select(fieldExpr)
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
	return interp.NewStruct(typ, fields), nil
}

func (sel *fieldSelectorArgument) Name() string {
	return sel.parent.Name() + "." + sel.fieldName
}

func (sel *fieldSelectorArgument) ValueFromContext(ctx *values.FuncInputs) (ir.Element, error) {
	val, err := sel.parent.ValueFromContext(ctx)
	if err != nil {
		return nil, err
	}
	structValue, ok := val.(interp.Selector)
	if !ok {
		return nil, errors.Errorf("%s is not a structure instance (type %s)", sel.parent.Name(), sel.parent.typ.String())
	}
	fieldVal, err := structValue.Select(&ir.SelectorExpr{
		Src: &ast.SelectorExpr{Sel: &ast.Ident{Name: sel.fieldName}},
	})
	if err != nil {
		return nil, err
	}
	if fieldVal == nil {
		return nil, errors.Errorf("no field %s in structure instance passed as argument %s (type %s)", sel.fieldName, sel.parent.Name(), sel.parent.typ.String())
	}
	return fieldVal, nil
}

type (
	sliceArgument struct {
		parentArgument
		typ   *ir.SliceType
		proxy interp.FixedSlice
	}

	indexSelectorArgument struct {
		parent *sliceArgument
		index  int
		proxy  ir.Element
	}
)

func (vis *inputVisitor) newSliceArgument(parent parentArgument, typ *ir.SliceType, proxy ir.Element) (*interp.Slice, error) {
	fixed, ok := proxy.(interp.FixedSlice)
	if !ok {
		return nil, errors.Errorf("%T does not support %s", proxy, reflect.TypeFor[interp.FixedSlice]().Name())
	}
	sliceArg := &sliceArgument{
		parentArgument: parent,
		typ:            typ,
		proxy:          fixed,
	}
	elType, ok := typ.ElementType()
	if !ok {
		return nil, errors.Errorf("atomic type %s cannot be indexed", typ.String())
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
	return interp.NewSlice(typ, args), nil
}

func (sel *indexSelectorArgument) Name() string {
	return sel.parent.Name() + "." + fmt.Sprintf("[%d]", sel.index)
}

func (sel *indexSelectorArgument) ValueFromContext(ctx *values.FuncInputs) (ir.Element, error) {
	el, err := sel.parent.ValueFromContext(ctx)
	if err != nil {
		return nil, err
	}
	sliceValue, ok := el.(*values.Slice)
	if !ok {
		return nil, errors.Errorf("%s is not a structure instance (type %s)", sel.parent.Name(), sel.parent.typ.String())
	}
	val := sliceValue.Element(sel.index)
	if val == nil {
		return nil, errors.Errorf("no element at %d in slice instance passed as argument %s (type %s)", sel.index, sel.parent.Name(), sel.parent.typ.String())
	}
	return val, nil
}

var _ parentArgument = (*indexSelectorArgument)(nil)

type arrayArgument struct {
	parentArgument
	typ ir.ArrayType

	file           *ir.File
	shape          *shape.Shape
	graphCallIndex int
	node           *BackendNode
}

var (
	_ processor.Argument              = (*arrayArgument)(nil)
	_ evaluator.NumericalElement      = (*arrayArgument)(nil)
	_ materialise.ElementMaterialiser = (*arrayArgument)(nil)
	_ flatten.Unflattener             = (*arrayArgument)(nil)
	_ interp.Slicer                   = (*arrayArgument)(nil)
	_ interp.WithAxes                 = (*arrayArgument)(nil)
)

// NewArrayArgument creates a new argument element that the graph can also use as an argument.
func (ev *Evaluator) NewArrayArgument(file *ir.File, parent parentArgument, typ ir.ArrayType, proxy ir.Element) (elements.ElementWithArrayFromContext, error) {
	vis := &inputVisitor{file: file, proc: ev.process}
	return vis.newArrayArgument(parent, typ, proxy)
}

// NewArrayArgument creates a new argument element that the graph can also use as an argument.
func (vis *inputVisitor) newArrayArgument(parent parentArgument, typ ir.ArrayType, proxy ir.Element) (elements.ElementWithArrayFromContext, error) {
	fixed, ok := proxy.(interp.FixedShape)
	if !ok {
		return nil, errors.Errorf("%T does not support %s", proxy, reflect.TypeFor[interp.FixedShape]().Name())
	}
	n := &arrayArgument{
		parentArgument: parent,
		typ:            typ,
		shape:          fixed.Shape(),
		file:           vis.file,
	}
	n.graphCallIndex = vis.proc.RegisterArg(n)
	return n, nil
}

// UnaryOp applies a unary operator on x.
func (n *arrayArgument) UnaryOp(ctx ir.Evaluator, expr *ir.UnaryExpr) (evaluator.NumericalElement, error) {
	node, err := n.materialise(ctx.(evaluator.Context).Materialiser())
	if err != nil {
		return nil, err
	}
	return node.UnaryOp(ctx, expr)
}

// BinaryOp applies a binary operator to x and y.
// Note that the receiver can be either the left or right argument.
func (n *arrayArgument) BinaryOp(ctx ir.Evaluator, expr *ir.BinaryExpr, x, y evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	node, err := n.materialise(ctx.(evaluator.Context).Materialiser())
	if err != nil {
		return nil, err
	}
	return node.BinaryOp(ctx, expr, x, y)
}

// Cast an element into a given data type.
func (n *arrayArgument) Cast(ctx ir.Evaluator, expr ir.AssignableExpr, target ir.Type) (evaluator.NumericalElement, error) {
	node, err := n.materialise(ctx.(evaluator.Context).Materialiser())
	if err != nil {
		return nil, err
	}
	return node.Cast(ctx, expr, target)
}

// Reshape an element.
func (n *arrayArgument) Reshape(ctx ir.Evaluator, expr ir.AssignableExpr, axisLengths []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	node, err := n.materialise(ctx.(evaluator.Context).Materialiser())
	if err != nil {
		return nil, err
	}
	return node.Reshape(ctx, expr, axisLengths)
}

func (n *arrayArgument) materialise(mat materialise.Materialiser) (*BackendNode, error) {
	if n.node != nil {
		return n.node, nil
	}
	op, err := mat.Graph().Core().Argument(n.Name(), n.shape, n.graphCallIndex)
	if err != nil {
		return nil, err
	}
	n.node, err = NewBackendNode(
		mat.(*arrayOps).ev,
		elements.NewExprAt(n.file,
			&ir.ValueRef{Stor: &ir.LocalVarStorage{
				Src: &ast.Ident{Name: n.Name()},
				Typ: n.typ,
			}},
		),
		&ops.OutputNode{
			Node:  op,
			Shape: n.shape,
		})
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
func (n *arrayArgument) Shape() *shape.Shape {
	return n.shape
}

func (n *arrayArgument) Axes(ev ir.Evaluator) (*interp.Slice, error) {
	return axesFromShape(ev, n.shape)
}

func (n *arrayArgument) ToDeviceHandle(dev platform.Device, in *values.FuncInputs) (platform.DeviceHandle, error) {
	array, err := n.ArrayFromContext(in)
	if err != nil {
		return nil, err
	}
	return toDevice(dev, array)
}

func toDevice(dev platform.Device, arr values.Array) (platform.DeviceHandle, error) {
	deviceArray, err := arr.ToDevice(dev)
	if err != nil {
		return nil, err
	}
	return deviceArray.DeviceHandle(), nil
}

// NumericalConstant returns the value of a constant represented by a node.
func (n *arrayArgument) ArrayFromContext(in *values.FuncInputs) (values.Array, error) {
	value, err := n.parentArgument.ValueFromContext(in)
	if err != nil {
		return nil, err
	}
	array, ok := value.(values.Array)
	if !ok {
		return nil, errors.Errorf("%s:%T is not an assigned numerical value", n.parentArgument.Name(), value)
	}
	return array, nil
}

func (n *arrayArgument) Copy() interp.Copier {
	return n
}

func (n *arrayArgument) Type() ir.Type {
	return n.typ
}

// Unflatten consumes the next handles to return a GX value.
func (n *arrayArgument) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseArray(n.typ)
}

// Slice of the value on the first axis given an index.
func (n *arrayArgument) Slice(fitp *interp.FileScope, expr *ir.IndexExpr, index evaluator.NumericalElement) (ir.Element, error) {
	node, err := n.materialise(fitp.Materialiser())
	if err != nil {
		return nil, err
	}
	return node.Slice(fitp, expr, index)
}

// SliceArray of the value on the first axis given an index.
func (n *arrayArgument) SliceArray(fitp *interp.FileScope, expr ir.AssignableExpr, index evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	node, err := n.materialise(fitp.Materialiser())
	if err != nil {
		return nil, err
	}
	return node.SliceArray(fitp, expr, index)
}
