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
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/interp/materialise"
)

// valueElement is a GX value represented as a node in the graph.
type valueElement struct {
	*BackendNode
	value *values.HostArray
}

var (
	_ elements.ElementWithConstant    = (*valueElement)(nil)
	_ interp.ArraySlicer              = (*valueElement)(nil)
	_ interp.Slicer                   = (*valueElement)(nil)
	_ materialise.ElementMaterialiser = (*valueElement)(nil)
	_ materialise.Node                = (*valueElement)(nil)
	_ interp.Copier                   = (*valueElement)(nil)
	_ interp.WithAxes                 = (*valueElement)(nil)
)

func newValueElement(ev *Evaluator, src elements.ExprAt, value values.Array) (*valueElement, error) {
	hostValue, err := value.ToHostArray(kernels.Allocator())
	if err != nil {
		return nil, err
	}
	cstNode, err := ev.ao.graph.Core().Constant(hostValue.Buffer())
	if err != nil {
		return nil, err
	}
	node, err := NewBackendNode(ev, src.ToExprAt(), &ops.OutputNode{
		Node:  cstNode,
		Shape: value.Shape(),
	})
	if err != nil {
		return nil, err
	}
	return &valueElement{
		BackendNode: node,
		value:       hostValue,
	}, nil
}

// NumericalConstant returns the value of a constant represented by a node.
func (n *valueElement) NumericalConstant() *values.HostArray {
	return n.value
}

// Unflatten creates a GX value from the next handles available in the parser.
func (n *valueElement) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return values.NewDeviceArray(n.value.Type(), handles.Next())
}

// Copy the graph node by returning itself.
func (n *valueElement) Copy() interp.Copier {
	return n
}

func (n *valueElement) Axes(ev ir.Evaluator) (*interp.Slice, error) {
	shape := n.value.Shape()
	ctx := ev.(evaluator.Context)
	axes := make([]ir.Element, len(shape.AxisLengths))
	for i, axisSize := range shape.AxisLengths {
		iExpr := &ir.AtomicValueT[ir.Int]{
			Val: ir.Int(i),
			Typ: ir.IntLenType(),
		}
		iValue, err := values.AtomIntegerValue[ir.Int](ir.IntLenType(), ir.Int(axisSize))
		if err != nil {
			return nil, err
		}
		axes[i], err = ctx.Evaluator().ElementFromAtom(ctx, iExpr, iValue)
		if err != nil {
			return nil, err
		}
	}
	return interp.NewSlice(ir.IntLenSliceType(), axes), nil
}

func (n *valueElement) Type() ir.Type {
	return n.value.Type()
}

func (n *valueElement) Kind() ir.Kind {
	return n.value.Type().Kind()
}

func (n *valueElement) String() string {
	return n.value.String()
}

// Materialise returns itself.
func (n *valueElement) Materialise(materialise.Materialiser) (materialise.Node, error) {
	return n, nil
}
