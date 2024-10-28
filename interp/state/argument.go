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

package state

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/state/proxies"
)

type (
	valueFromCallFetcher interface {
		valueProxy(receiver proxies.Value, args []proxies.Value) proxies.Value
		value(Context) values.Value
		String() string
	}

	callArgumentFetcher struct {
		argIndex int
	}

	receiverFetcher struct{}
)

func (f callArgumentFetcher) value(ctx Context) values.Value {
	return ctx.Args()[f.argIndex]
}

func (f callArgumentFetcher) valueProxy(receiver proxies.Value, args []proxies.Value) proxies.Value {
	return args[f.argIndex]
}

func (f callArgumentFetcher) String() string {
	return fmt.Sprintf("GX argument %d", f.argIndex)
}

func (f receiverFetcher) value(ctx Context) values.Value {
	return ctx.Receiver()
}

func (f receiverFetcher) valueProxy(receiver proxies.Value, args []proxies.Value) proxies.Value {
	return receiver
}

func (f receiverFetcher) String() string {
	return "GX receiver"
}

// ArgGX is an argument passed to the GX interpreter from the host language.
type ArgGX struct {
	state        *State
	field        ExprFile[*ir.Field]
	traceValue   proxies.Value
	valueFetcher valueFromCallFetcher

	// backendArg is used when the value represented by the argument
	// is a numerical used as such in the backend graph.
	backendArg *argIdentity
}

var (
	_ Slicer                  = (*ArgGX)(nil)
	_ backendElement          = (*ArgGX)(nil)
	_ ElementWithValueContext = (*ArgGX)(nil)
)

// ArgGX represents a GX argument.
func (g *State) ArgGX(field FieldAt, index int, args []proxies.Value) Element {
	return g.newArg(field, callArgumentFetcher{
		argIndex: index,
	}, nil, args)
}

// Receiver represents a GX function call receiver.
func (g *State) Receiver(field FieldAt, recv proxies.Value) Element {
	return g.newArg(field, receiverFetcher{}, recv, nil)
}

func (g *State) newArg(field FieldAt, valueFetcher valueFromCallFetcher, recv proxies.Value, args []proxies.Value) Element {
	n := &ArgGX{
		state:        g,
		field:        field,
		valueFetcher: valueFetcher,
	}
	n.traceValue = n.valueFetcher.valueProxy(recv, args)
	structValue, ok := n.traceValue.(*proxies.Struct)
	if !ok {
		return n
	}
	return g.StructWithInit(structValue.StructType(), field.ToExprAt(), n)
}

// State owning the element.
func (n *ArgGX) State() *State {
	return n.state
}

// SelectField constructs a field given its ID.
func (n *ArgGX) SelectField(expr ExprAt, id int) (Element, error) {
	structValue, ok := n.traceValue.(*proxies.Struct)
	if !ok {
		return nil, errors.Errorf("argument %s: cannot convert %T to platform.Struct", n.Name(), n.traceValue)
	}
	return n.state.newArgField(n, structValue, expr, id)
}

// Slice constructs a slice given an index.
func (n *ArgGX) Slice(expr ExprAt, i int) (Element, error) {
	switch valT := n.traceValue.(type) {
	case *proxies.Slice:
		return n.state.newArgSliceElement(n, valT, expr, i)
	case *proxies.Numerical:
		if _, err := n.nodes(); err != nil {
			return nil, err
		}
		slice, err := n.state.backendGraph.Core().NewSlice(n.backendArg.op.Node, i)
		if err != nil {
			return nil, err
		}
		return n.state.ElementFromNode(expr, slice, &shape.Shape{
			DType:       n.Shape().DType,
			AxisLengths: n.Shape().AxisLengths[1:],
		})
	default:
		return nil, errors.Errorf("argument %s: cannot convert %T to platform.Slice", n.Name(), n.traceValue)
	}
}

// Type of the static variable.
func (n *ArgGX) Type() ir.Type {
	return n.field.Type()
}

// String returns a string representation of the node.
func (n *ArgGX) String() string {
	return fmt.Sprintf("value %s:%s", n.Name(), n.Type().String())
}

// Name returns the name of the argument.
func (n *ArgGX) Name() string {
	return n.field.ExprT().Name.Name
}

// Expr returns the field expression declaring the argument.
func (n *ArgGX) Expr() ir.Expr {
	return n.field.ExprT()
}

// Field returns the field declaring the argument.
func (n *ArgGX) Field() FieldAt {
	return n.field
}

func (n *ArgGX) nodes() ([]*graph.OutputNode, error) {
	if n.backendArg != nil {
		return []*graph.OutputNode{n.backendArg.op}, nil
	}
	var err error
	n.backendArg, err = n.state.newIdentity(n, n.field.ToExprAt())
	if err != nil {
		return nil, err
	}
	return n.backendArg.nodes()
}

// NumericalConstant returns the value of a constant represented by a node.
func (n *ArgGX) ValueFromContext(ctx Context) (values.Array, error) {
	if _, err := n.nodes(); err != nil {
		return nil, err
	}
	return n.backendArg.ValueFromContext(ctx)
}

func (n *ArgGX) valueFromHandle(handles *handleParser) (values.Value, error) {
	return values.NewDeviceArray(n.Type(), handles.next()), nil
}

// Shape of the value of the argument.
func (n *ArgGX) Shape() *shape.Shape {
	return n.traceValue.(*proxies.Numerical).Shape()
}

// Value returns the value of an argument given an interpreter context.
func (n *ArgGX) Value(ctx Context) values.Value {
	return n.valueFetcher.value(ctx)
}

// argBackend is a helper to build backend argument.
type argBackend struct {
	state *State
	expr  ExprAt
	name  string
	index int
}

func (a argBackend) Type() ir.Type {
	return a.expr.Type()
}

func (a argBackend) Index() int {
	return a.index
}

// Name of the argument.
func (a argBackend) Name() string {
	return a.name
}

// State owning the element.
func (a argBackend) State() *State {
	return a.state
}

func (a *argBackend) valueFromHandle(handles *handleParser) (values.Value, error) {
	return values.NewDeviceArray(a.expr.Type(), handles.next()), nil
}

// argIdentity is an argument passing a GX argument directly to the backend without modifications.
type argIdentity struct {
	argBackend
	op   *graph.OutputNode
	orig *ArgGX
}

var (
	_ Element                 = (*argIdentity)(nil)
	_ ElementWithValueContext = (*ArgGX)(nil)
)

// NewArgConst returns a new constant argument passed to the backend.
func (g *State) newIdentity(orig *ArgGX, expr ExprAt) (*argIdentity, error) {
	value, ok := orig.traceValue.(*proxies.Numerical)
	if !ok {
		return nil, errors.Errorf("cannot convert argument %s to a backend argument: %T cannot be converted to platform.Numerical", orig.Name(), orig.traceValue)
	}
	arg := &argIdentity{
		argBackend: argBackend{
			state: g,
			expr:  expr,
			name:  orig.Name(),
		},
		orig: orig,
	}
	arg.index = g.RegisterArg(arg)
	op, err := g.backendGraph.Core().NewArgument(arg.orig.Name(), value.Shape(), arg.index)
	if err != nil {
		return nil, err
	}
	arg.op = &graph.OutputNode{Node: op, Shape: value.Shape()}
	return arg, nil
}

func (a *argIdentity) Init(args []values.Value) error {
	return nil
}

func (a *argIdentity) ToDeviceHandle(device platform.Device, ctx Context) (platform.DeviceHandle, error) {
	value := a.orig.Value(ctx)
	array, ok := value.(values.Array)
	if !ok {
		return nil, errors.Errorf("%s:%s:%T is not an assigned numerical value", a.orig.valueFetcher, a.orig.Name(), value)
	}
	return toDevice(device, array)
}

func (a *argIdentity) ValueFromContext(ctx Context) (values.Array, error) {
	arg := a.orig.valueFetcher.value(ctx)
	array, ok := arg.(values.Array)
	if !ok {
		return nil, errors.Errorf("cannot cast argument %s (%T) to %s", a.Name(), arg, reflect.TypeFor[*argIdentity]())
	}
	return array, nil
}

func (a *argIdentity) Numerical() values.Array {
	return a.orig.traceValue.(values.Array)
}

func (a *argIdentity) Shape() *shape.Shape {
	return a.orig.traceValue.(*proxies.Numerical).Shape()
}

func (a *argIdentity) nodes() ([]*graph.OutputNode, error) {
	return []*graph.OutputNode{a.op}, nil
}

// argElement is a constant argument passed to the backend.
type argElement struct {
	argBackend
	Argument
	op *graph.OutputNode
}

var _ backendElement = (*argElement)(nil)

// NewArgElement returns a new state element from an argument.
func (g *State) NewArgElement(name string, expr ExprAt, arg Argument) (NumericalElement, error) {
	elmt := &argElement{
		argBackend: argBackend{
			state: g,
			expr:  expr,

			name: name,
		},
		Argument: arg,
	}
	elmt.index = g.RegisterArg(arg)
	nod, err := g.backendGraph.Core().NewArgument(elmt.name, arg.Shape(), elmt.index)
	if err != nil {
		return nil, err
	}
	elmt.op = &graph.OutputNode{
		Shape: arg.Shape(),
		Node:  nod,
	}
	return elmt, nil
}

func (a *argElement) nodes() ([]*graph.OutputNode, error) {
	return []*graph.OutputNode{a.op}, nil
}

func (g *State) newArgSliceElement(parent fieldInArgSelector, sliceValue *proxies.Slice, expr ExprAt, id int) (Element, error) {
	return g.newSubArg(parent, fmt.Sprintf("[%d]", id), id, expr, sliceValue.Element(id))
}

func (g *State) newArgField(parent fieldInArgSelector, structValue *proxies.Struct, expr ExprAt, id int) (Element, error) {
	fieldValues := structValue.Values()
	structType := structValue.StructType()
	structFields := structType.Fields.Fields()
	field := structFields[id]
	return g.newSubArg(parent, field.Name.Name, id, expr, fieldValues[id])

}

func (g *State) newSubArg(parent fieldInArgSelector, name string, id int, expr ExprAt, value proxies.Value) (Element, error) {
	switch valueT := value.(type) {
	case *proxies.Numerical:
		return g.newArrayFieldArgument(parent, name, id, expr, valueT)
	case *proxies.Struct:
		return g.newStructFieldArgument(parent, name, id, expr, valueT), nil
	case *proxies.Slice:
		return g.newSliceFieldArgument(parent, name, id, expr, valueT), nil
	}
	return nil, errors.Errorf("argument name %s: field %d of type %T not supported", parent.Name(), id, value)
}

func (n *ArgGX) selectFieldInArgs(ctx Context, id int) (values.Value, error) {
	argValue := n.valueFetcher.value(ctx)
	switch argValueT := argValue.(type) {
	case *values.Struct:
		return argValueT.FieldValue(id), nil
	case *values.Slice:
		return argValueT.Element(id), nil
	default:
		return nil, errors.Errorf("fetch id %d in argument %s:%T not supported", id, n.Name(), argValue)
	}
}

type (
	fieldInArgSelector interface {
		selectFieldInArgs(ctx Context, id int) (values.Value, error)
		Name() string
		Type() ir.Type
	}

	arrayFieldArgument struct {
		argBackend
		op         *graph.OutputNode
		parent     fieldInArgSelector
		fieldIndex int
		traceValue *proxies.Numerical
	}
)

var (
	_ Slicer                  = (*arrayFieldArgument)(nil)
	_ ElementWithValueContext = (*arrayFieldArgument)(nil)
	_ backendElement          = (*arrayFieldArgument)(nil)
)

func (g *State) newArrayFieldArgument(parent fieldInArgSelector, name string, index int, expr ExprAt, value *proxies.Numerical) (NumericalElement, error) {
	elmt := arrayFieldArgument{
		argBackend: argBackend{
			state: g,
			expr:  expr,
			name:  name,
		},
		parent:     parent,
		fieldIndex: index,
		traceValue: value,
	}
	elmt.index = g.RegisterArg(&elmt)
	op, err := g.backendGraph.Core().NewArgument(elmt.name, value.Shape(), elmt.index)
	if err != nil {
		return nil, err
	}
	elmt.op = &graph.OutputNode{
		Node:  op,
		Shape: value.Shape(),
	}
	return &elmt, nil
}

func (a *arrayFieldArgument) Init(args []values.Value) error {
	return nil
}

func (a *arrayFieldArgument) ToDeviceHandle(device platform.Device, ctx Context) (platform.DeviceHandle, error) {
	value, err := a.parent.selectFieldInArgs(ctx, a.fieldIndex)
	if err != nil {
		return nil, err
	}
	array, ok := value.(values.Array)
	if !ok {
		return nil, errors.Errorf("GX argument %s:%T is not an assigned numerical value", a.name, value)
	}
	return toDevice(device, array)
}

func (a *arrayFieldArgument) ValueFromContext(ctx Context) (values.Array, error) {
	value, err := a.parent.selectFieldInArgs(ctx, a.fieldIndex)
	if err != nil {
		return nil, err
	}
	array, ok := value.(values.Array)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", value, reflect.TypeFor[*values.HostArray]().String())
	}
	return array, nil
}

func (a *arrayFieldArgument) Slice(expr ExprAt, i int) (Element, error) {
	sliceNode, err := a.state.backendGraph.Core().NewSlice(a.op.Node, i)
	if err != nil {
		return nil, err
	}
	return a.state.ElementFromNode(expr, sliceNode, &shape.Shape{
		DType:       a.traceValue.Shape().DType,
		AxisLengths: a.traceValue.Shape().AxisLengths[1:],
	})
}

func (a *arrayFieldArgument) nodes() ([]*graph.OutputNode, error) {
	return []*graph.OutputNode{a.op}, nil
}

func (a *arrayFieldArgument) Shape() *shape.Shape {
	return a.traceValue.Shape()
}

func (a *arrayFieldArgument) Name() string {
	return a.parent.Name() + "." + a.name
}

func (a *arrayFieldArgument) String() string {
	return a.Name()
}

type structFieldArgument struct {
	argBackend
	parent     fieldInArgSelector
	fieldIndex int
	traceValue *proxies.Struct
}

var _ fieldInArgSelector = (*structFieldArgument)(nil)

func (g *State) newStructFieldArgument(parent fieldInArgSelector, name string, index int, expr ExprAt, value *proxies.Struct) Element {
	selector := &structFieldArgument{
		argBackend: argBackend{
			state: g,
			expr:  expr,
			name:  name,
		},
		parent:     parent,
		fieldIndex: index,
		traceValue: value,
	}
	return g.StructWithInit(value.StructType(), expr, selector)
}

func (a *structFieldArgument) selectFieldInArgs(ctx Context, id int) (values.Value, error) {
	parentValue, err := a.parent.selectFieldInArgs(ctx, a.fieldIndex)
	if err != nil {
		return nil, err
	}
	return parentValue.(*values.Struct).FieldValue(id), nil
}

func (a *structFieldArgument) SelectField(expr ExprAt, id int) (Element, error) {
	return a.state.newArgField(a, a.traceValue, expr, id)
}

func (a *structFieldArgument) Name() string {
	return a.parent.Name() + "." + a.name
}

func (a *structFieldArgument) String() string {
	return a.Name()
}

type sliceFieldArgument struct {
	argBackend
	parent     fieldInArgSelector
	fieldIndex int
	traceValue *proxies.Slice
}

func (g *State) newSliceFieldArgument(parent fieldInArgSelector, name string, index int, expr ExprAt, value *proxies.Slice) Element {
	selector := &sliceFieldArgument{
		argBackend: argBackend{
			state: g,
			expr:  expr,
			name:  name,
		},
		parent:     parent,
		fieldIndex: index,
		traceValue: value,
	}
	return g.Slice(expr.ToExprAt(), selector, value.Size())
}

func (a *sliceFieldArgument) SelectField(expr ExprAt, id int) (Element, error) {
	return a.state.newArgSliceElement(a, a.traceValue, expr, id)
}

func (a *sliceFieldArgument) selectFieldInArgs(ctx Context, id int) (values.Value, error) {
	parentValue, err := a.parent.selectFieldInArgs(ctx, a.fieldIndex)
	if err != nil {
		return nil, err
	}
	return parentValue.(*values.Slice).Element(id), nil
}
