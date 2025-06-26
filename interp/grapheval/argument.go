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

	"github.com/pkg/errors"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/proxies"
)

type (
	argFetcher interface {
		ValueProxy() proxies.Value
		ValueFromContext(*elements.InputValues) (values.Value, error)
	}

	parameterFetcher struct {
		pValue     proxies.Value
		paramIndex int
	}

	receiverFetcher struct {
		pValue proxies.Value
	}
)

func (f parameterFetcher) ValueFromContext(in *elements.InputValues) (values.Value, error) {
	return in.Args[f.paramIndex], nil
}

func (f parameterFetcher) ValueProxy() proxies.Value {
	return f.pValue
}

func (f parameterFetcher) String() string {
	return fmt.Sprintf("GX argument %d", f.paramIndex)
}

func (f receiverFetcher) ValueFromContext(in *elements.InputValues) (values.Value, error) {
	return in.Receiver, nil
}

func (f receiverFetcher) ValueProxy() proxies.Value {
	return f.pValue
}

func (f receiverFetcher) String() string {
	return "GX receiver"
}

// ArgGX represents a GX argument.
func (ev *Evaluator) ArgGX(field elements.FieldAt, index int, args []proxies.Value) (elements.Element, error) {
	return ev.newRootArg(field, parameterFetcher{
		paramIndex: index,
		pValue:     args[index],
	})
}

// Receiver represents a GX function call receiver.
func (ev *Evaluator) Receiver(field elements.FieldAt, recv proxies.Value) (elements.Element, error) {
	return ev.newRootArg(field, receiverFetcher{
		pValue: recv,
	})
}

type (
	parentArgument interface {
		Name() string
		ValueProxy() proxies.Value
		ValueFromContext(*elements.InputValues) (values.Value, error)
	}

	// rootArgument is an argument passed to the GX interpreter from the host language.
	rootArgument struct {
		argFetcher
		field elements.NodeFile[*ir.Field]
	}
)

var _ parentArgument = (*rootArgument)(nil)

func (ev *Evaluator) newRootArg(field elements.FieldAt, fetcher argFetcher) (elements.Element, error) {
	n := &rootArgument{
		argFetcher: fetcher,
		field:      field,
	}
	return ev.newArg(n, elements.NewExprAt(field.File(), &ir.ValueRef{
		Src:  field.Node().Name,
		Stor: field.Node().Storage(),
	}))
}

func (ev *Evaluator) newArg(parent parentArgument, expr elements.ExprAt) (elements.Element, error) {
	switch pValueT := parent.ValueProxy().(type) {
	case *proxies.Array:
		return ev.NewArrayArgument(parent, expr, pValueT)
	case *proxies.NamedType:
		return ev.newNamedTypeArgument(parent, expr, pValueT)
	case *proxies.Struct:
		return ev.newStructArgument(parent, expr, pValueT)
	case *proxies.Slice:
		return ev.newSliceArgument(parent, expr, pValueT)
	default:
		return nil, errors.Errorf("argument type %T not supported", pValueT)
	}
}

func (a *rootArgument) Name() string {
	return a.field.Node().Name.Name
}

type namedTypeArgument struct {
	parent parentArgument
	pValue *proxies.NamedType
}

func (ev *Evaluator) newNamedTypeArgument(parent parentArgument, expr elements.ExprAt, pValue *proxies.NamedType) (*elements.NamedType, error) {
	arg := &namedTypeArgument{
		parent: parent,
		pValue: pValue,
	}
	recv, err := ev.newArg(arg, expr)
	if err != nil {
		return nil, err
	}
	recvCopier, ok := recv.(elements.Copier)
	if !ok {
		return nil, errors.Errorf("element %T cannot be used as a receiver", recv)
	}
	return elements.NewNamedType(ev.NewFunc, arg.pValue.NamedType(), recvCopier), nil
}

func (arg *namedTypeArgument) Name() string {
	return arg.parent.Name() + "." + arg.pValue.NamedType().Name()
}

func (arg *namedTypeArgument) ValueProxy() proxies.Value {
	return arg.pValue.Under()
}

func (arg *namedTypeArgument) ValueFromContext(ctx *elements.InputValues) (values.Value, error) {
	return arg.parent.ValueFromContext(ctx)
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

var _ parentArgument = (*fieldSelectorArgument)(nil)

func (ev *Evaluator) newStructArgument(parent parentArgument, expr elements.ExprAt, pValue *proxies.Struct) (*elements.Struct, error) {
	structType := pValue.StructType()
	structArg := &structArgument{
		parentArgument: parent,
		pValue:         pValue,
	}
	fields := make(map[string]elements.Element, structType.NumFields())
	for _, field := range structType.Fields.Fields() {
		name := field.Name.Name
		fieldValue := pValue.Field(name)
		if fieldValue == nil {
			return nil, errors.Errorf("no field %s in structure instance passed as argument %s (type %s)", name, parent.Name(), pValue.Type().String())
		}
		selector := &fieldSelectorArgument{
			parent:    structArg,
			fieldName: name,
			pValue:    fieldValue,
		}
		fieldExpr := elements.NewExprAt(expr.File(), &ir.ValueRef{
			Src: &ast.Ident{
				NamePos: expr.Node().Source().Pos(),
				Name:    selector.Name(),
			},
			Stor: &ir.FieldStorage{Field: field},
		})
		field, err := ev.newArg(selector, fieldExpr)
		if err != nil {
			return nil, err
		}
		fields[name] = field
	}
	return elements.NewStruct(pValue.StructType(), expr.ToValueAt(), fields), nil
}

func (sel *fieldSelectorArgument) Name() string {
	return sel.parent.Name() + "." + sel.fieldName
}

func (sel *fieldSelectorArgument) ValueProxy() proxies.Value {
	return sel.pValue
}

func (sel *fieldSelectorArgument) ValueFromContext(ctx *elements.InputValues) (values.Value, error) {
	val, err := sel.parent.ValueFromContext(ctx)
	if err != nil {
		return nil, err
	}
	structValue, ok := values.Underlying(val).(*values.Struct)
	if !ok {
		return nil, errors.Errorf("%s is not a structure instance (type %s)", sel.parent.Name(), sel.parent.pValue.Type().String())
	}
	fieldVal := structValue.FieldValue(sel.fieldName)
	if fieldVal == nil {
		return nil, errors.Errorf("no field %s in structure instance passed as argument %s (type %s)", sel.fieldName, sel.parent.Name(), sel.parent.pValue.Type().String())
	}
	return fieldVal, nil
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

func (ev *Evaluator) newSliceArgument(parent parentArgument, expr elements.ExprAt, pValue *proxies.Slice) (*elements.Slice, error) {
	sliceArg := &sliceArgument{
		parentArgument: parent,
		pValue:         pValue,
	}
	vals := make([]elements.Element, pValue.Size())
	slicerType, ok := expr.Node().Type().(ir.SlicerType)
	if !ok {
		return nil, errors.Errorf("type %s cannot be indexed", expr.Node().Type().String())
	}
	elType, ok := slicerType.ElementType()
	if !ok {
		return nil, errors.Errorf("atomic type %s cannot be indexed", slicerType.String())
	}
	for i := range vals {
		valI, err := pValue.Element(i)
		if err != nil {
			return nil, err
		}
		selector := &indexSelectorArgument{
			parent: sliceArg,
			index:  i,
			pValue: valI,
		}
		src := &ast.Ident{
			NamePos: expr.Node().Source().Pos(),
			Name:    fmt.Sprintf("[%d]", i),
		}
		valExpr := elements.NewExprAt(expr.File(), &ir.IndexExpr{
			Src: &ast.IndexExpr{
				X:     expr.Node().Source().(ast.Expr),
				Index: src,
			},
			X: expr.ToExprAt().Node(),
			Index: &ir.AtomicValueT[ir.Int]{
				Src: src,
				Val: ir.Int(i),
				Typ: ir.DefaultIntType,
			},
			Typ: elType,
		})
		vals[i], err = ev.newArg(selector, valExpr)
		if err != nil {
			return nil, err
		}

	}
	return elements.NewSlice(expr, vals), nil
}

func (sel *indexSelectorArgument) Name() string {
	return sel.parent.Name() + "." + fmt.Sprintf("[%d]", sel.index)
}

func (sel *indexSelectorArgument) ValueProxy() proxies.Value {
	return sel.pValue
}

func (sel *indexSelectorArgument) ValueFromContext(ctx *elements.InputValues) (values.Value, error) {
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

var _ parentArgument = (*indexSelectorArgument)(nil)

type arrayArgument struct {
	*BackendNode
	parentArgument
	pValue         *proxies.Array
	graphCallIndex int
	expr           elements.ExprAt
}

var (
	_ processor.Argument = (*arrayArgument)(nil)
)

// NewArrayArgument creates a new argument element that the graph can also use as an argument.
func (ev *Evaluator) NewArrayArgument(parent parentArgument, expr elements.ExprAt, pValue *proxies.Array) (elements.ElementWithArrayFromContext, error) {
	n := &arrayArgument{
		parentArgument: parent,
		expr:           expr,
		pValue:         pValue,
	}
	n.graphCallIndex = ev.Processor().RegisterArg(n)
	op, err := ev.ArrayOps().Graph().Core().Argument(n.Name(), n.pValue.Shape(), n.graphCallIndex)
	if err != nil {
		return nil, err
	}
	n.BackendNode, err = ElementFromNode(expr, &ops.OutputNode{
		Node:  op,
		Shape: pValue.Shape(),
	})
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (n *arrayArgument) ToDeviceHandle(dev platform.Device, in *elements.InputValues) (platform.DeviceHandle, error) {
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
func (n *arrayArgument) ArrayFromContext(in *elements.InputValues) (values.Array, error) {
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

func (n *arrayArgument) Copy() elements.Copier {
	return n
}

func (*arrayArgument) Kind() ir.Kind {
	return ir.ArrayKind
}
