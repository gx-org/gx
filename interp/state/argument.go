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

	"github.com/pkg/errors"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/state/proxies"
)

type (
	argFetcher interface {
		ValueProxy() proxies.Value
		ValueFromContext(Context) (values.Value, error)
	}

	parameterFetcher struct {
		pValue     proxies.Value
		paramIndex int
	}

	receiverFetcher struct {
		pValue proxies.Value
	}
)

func (f parameterFetcher) ValueFromContext(ctx Context) (values.Value, error) {
	return ctx.Args()[f.paramIndex], nil
}

func (f parameterFetcher) ValueProxy() proxies.Value {
	return f.pValue
}

func (f parameterFetcher) String() string {
	return fmt.Sprintf("GX argument %d", f.paramIndex)
}

func (f receiverFetcher) ValueFromContext(ctx Context) (values.Value, error) {
	return ctx.Receiver(), nil
}

func (f receiverFetcher) ValueProxy() proxies.Value {
	return f.pValue
}

func (f receiverFetcher) String() string {
	return "GX receiver"
}

// ArgGX represents a GX argument.
func (g *State) ArgGX(field elements.FieldAt, index int, args []proxies.Value) (Element, error) {
	return g.newRootArg(field, parameterFetcher{
		paramIndex: index,
		pValue:     args[index],
	})
}

// Receiver represents a GX function call receiver.
func (g *State) Receiver(field elements.FieldAt, recv proxies.Value) (Element, error) {
	return g.newRootArg(field, receiverFetcher{
		pValue: recv,
	})
}

type (
	parentArgument interface {
		State() *State
		Name() string
		ValueProxy() proxies.Value
		ValueFromContext(Context) (values.Value, error)
	}

	// rootArgument is an argument passed to the GX interpreter from the host language.
	rootArgument struct {
		argFetcher
		state *State
		field elements.NodeFile[*ir.Field]
	}
)

var _ parentArgument = (*rootArgument)(nil)

func (g *State) newRootArg(field elements.FieldAt, fetcher argFetcher) (Element, error) {
	n := &rootArgument{
		argFetcher: fetcher,
		state:      g,
		field:      field,
	}
	return g.newArg(n, field.ToExprAt())
}

func (g *State) newArg(parent parentArgument, expr elements.ExprAt) (Element, error) {
	switch pValueT := parent.ValueProxy().(type) {
	case *proxies.Array:
		return NewArrayArgument(parent, expr, pValueT)
	case *proxies.Struct:
		return newStructArgument(parent, expr, pValueT)
	case *proxies.Slice:
		return newSliceArgument(parent, expr, pValueT)
	default:
		return nil, errors.Errorf("argument type %T not supported", pValueT)
	}
}

func (a *rootArgument) State() *State {
	return a.state
}

func (a *rootArgument) Name() string {
	return a.field.Node().Name.Name
}

type (
	structArgument struct {
		parentArgument
		pValue *proxies.Struct
	}

	fieldSelectorArgument struct {
		parent    *structArgument
		fieldName string
		pValue    proxies.Value
	}
)

var (
	_ FieldSelector  = (*structArgument)(nil)
	_ parentArgument = (*fieldSelectorArgument)(nil)
)

func newStructArgument(parent parentArgument, expr elements.ExprAt, pValue *proxies.Struct) (*Struct, error) {
	return parent.State().StructWithInit(pValue.StructType(), expr, &structArgument{
		parentArgument: parent,
		pValue:         pValue,
	}), nil
}

func (n *structArgument) SelectField(expr elements.ExprAt, name string) (Element, error) {
	pValue := n.pValue.Field(name)
	if pValue == nil {
		return nil, errors.Errorf("no field %s in structure instance passed as argument %s (type %s)", name, n.Name(), n.pValue.Type().String())
	}
	return n.State().newArg(&fieldSelectorArgument{
		parent:    n,
		fieldName: name,
		pValue:    pValue,
	}, expr)
}

func (sel *fieldSelectorArgument) State() *State {
	return sel.parent.State()
}

func (sel *fieldSelectorArgument) Name() string {
	return sel.parent.Name() + "." + sel.fieldName
}

func (sel *fieldSelectorArgument) ValueProxy() proxies.Value {
	return sel.pValue
}

func (sel *fieldSelectorArgument) ValueFromContext(ctx Context) (values.Value, error) {
	el, err := sel.parent.ValueFromContext(ctx)
	if err != nil {
		return nil, err
	}
	structValue, ok := el.(*values.Struct)
	if !ok {
		return nil, errors.Errorf("%s is not a structure instance (type %s)", sel.parent.Name(), sel.parent.pValue.Type().String())
	}
	val := structValue.FieldValue(sel.fieldName)
	if val == nil {
		return nil, errors.Errorf("no field %s in structure instance passed as argument %s (type %s)", sel.fieldName, sel.parent.Name(), sel.parent.pValue.Type().String())
	}
	return val, nil
}

type (
	sliceArgument struct {
		parentArgument
		pValue *proxies.Slice
	}

	indexSelectorArgument struct {
		parent *sliceArgument
		index  int
		pValue proxies.Value
	}
)

func newSliceArgument(parent parentArgument, expr elements.ExprAt, pValue *proxies.Slice) (*Slice, error) {
	return parent.State().Slice(expr, &sliceArgument{
		parentArgument: parent,
		pValue:         pValue,
	}, pValue.Size()), nil
}

func (n *sliceArgument) Slice(expr elements.ExprAt, index int) (Element, error) {
	pValue, err := n.pValue.Element(index)
	if err != nil {
		return nil, err
	}
	return n.State().newArg(&indexSelectorArgument{
		parent: n,
		index:  index,
		pValue: pValue,
	}, expr)
}

func (sel *indexSelectorArgument) State() *State {
	return sel.parent.State()
}

func (sel *indexSelectorArgument) Name() string {
	return sel.parent.Name() + "." + fmt.Sprintf("[%d]", sel.index)
}

func (sel *indexSelectorArgument) ValueProxy() proxies.Value {
	return sel.pValue
}

func (sel *indexSelectorArgument) ValueFromContext(ctx Context) (values.Value, error) {
	el, err := sel.parent.ValueFromContext(ctx)
	if err != nil {
		return nil, err
	}
	sliceValue, ok := el.(*values.Slice)
	if !ok {
		return nil, errors.Errorf("%s is not a structure instance (type %s)", sel.parent.Name(), sel.parent.pValue.Type().String())
	}
	val := sliceValue.Element(sel.index)
	if val == nil {
		return nil, errors.Errorf("no element at %d in slice instance passed as argument %s (type %s)", sel.index, sel.parent.Name(), sel.parent.pValue.Type().String())
	}
	return val, nil
}

var (
	_ Slicer         = (*sliceArgument)(nil)
	_ parentArgument = (*indexSelectorArgument)(nil)
)

type arrayArgument struct {
	parentArgument
	pValue         *proxies.Array
	graphCallIndex int
	expr           elements.ExprAt
	node           *graph.OutputNode
}

var (
	_ Argument     = (*arrayArgument)(nil)
	_ Materialiser = (*arrayArgument)(nil)
	_ Slicer       = (*arrayArgument)(nil)
)

// NewArrayArgument creates a new argument element that the graph can also use as an argument.
func NewArrayArgument(parent parentArgument, expr elements.ExprAt, pValue *proxies.Array) (ElementWithArrayFromContext, error) {
	n := &arrayArgument{
		parentArgument: parent,
		expr:           expr,
		pValue:         pValue,
	}
	n.graphCallIndex = n.State().RegisterArg(n)
	op, err := n.State().backendGraph.Core().NewArgument(n.Name(), n.pValue.Shape(), n.graphCallIndex)
	if err != nil {
		return nil, err
	}
	n.node = &graph.OutputNode{
		Node:  op,
		Shape: n.pValue.Shape(),
	}
	return n, nil
}

func (n *arrayArgument) Flatten() ([]Element, error) {
	return []Element{n}, nil
}

func (n *arrayArgument) Materialise() (*graph.OutputNode, error) {
	return n.node, nil
}

func (n *arrayArgument) Shape() *shape.Shape {
	return n.pValue.Shape()
}

func (n *arrayArgument) ToDeviceHandle(dev *api.Device, ctx Context) (platform.DeviceHandle, error) {
	array, err := n.ArrayFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return toDevice(dev, array)
}

// NumericalConstant returns the value of a constant represented by a node.
func (n *arrayArgument) ArrayFromContext(ctx Context) (values.Array, error) {
	value, err := n.parentArgument.ValueFromContext(ctx)
	if err != nil {
		return nil, err
	}
	array, ok := value.(values.Array)
	if !ok {
		return nil, errors.Errorf("%s:%T is not an assigned numerical value", n.parentArgument.Name(), value)
	}
	return array, nil
}

func (n *arrayArgument) Slice(expr elements.ExprAt, i int) (Element, error) {
	sliceNode, err := n.State().backendGraph.Core().NewSlice(n.node.Node, i)
	if err != nil {
		return nil, err
	}
	return n.State().ElementFromNode(expr, &graph.OutputNode{
		Node: sliceNode,
		Shape: &shape.Shape{
			DType:       n.pValue.Shape().DType,
			AxisLengths: n.pValue.Shape().AxisLengths[1:],
		},
	})
}

func (n *arrayArgument) valueFromHandle(handles *handleParser) (values.Value, error) {
	return n.ArrayFromContext(handles.context())
}
