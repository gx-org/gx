// Copyright 2025 Google LLC
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

// Package eagereval implements the eager evaluation of GX functions.
package eagereval

import (
	"unsafe"

	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/concrete"
	"github.com/gx-org/gx/internal/interp/constants"
	"github.com/gx-org/gx/internal/interp/numbers"
	"github.com/gx-org/gx/internal/togo"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/interp/materialise"
)

type (
	// NumericalNode is a numerical element holding a reference to a backend value.
	NumericalNode interface {
		engine.NumericalElement
		materialise.Node
		hostio.Value
	}

	// EagerBackend is an interface implemented by backend only supporting eager mode.
	EagerBackend interface {
		ToElement(compute.Value, ir.Type) (NumericalNode, error)
	}
)

// Ops executes operations eagerly with a backend function.
type Ops struct {
	bck   EagerBackend
	graph compute.Function
}

var (
	_ engine.Engine            = (*Ops)(nil)
	_ materialise.Materialiser = (*Ops)(nil)
	_ hostio.InputConverter    = (*Ops)(nil)
)

func newRegistry(bck EagerBackend, importer ir.Importer, fn compute.Function) (*engine.Registry, *Ops) {
	ao := &Ops{bck: bck, graph: fn}
	proc := processor.New()
	return &engine.Registry{
		Importer:       importer,
		CompEvalEngine: ao,
		HostEngine:     ao,
		Processor:      proc,
		Runners:        interp.Runners(),
		Trace:          proc.RegisterTrace,
	}, ao
}

// Graph returns the backend function executing the operations.
func (ao *Ops) Graph() compute.Function {
	return ao.graph
}

// SubGraph returns an error: sub-graphs are not supported when operations are executed eagerly.
func (ao *Ops) SubGraph(name string) (engine.Engine, error) {
	return nil, errors.Errorf("cannot create sub-graph %s: not supported in eager mode", name)
}

func (ao *Ops) toElement(ev ir.Evaluator, val compute.Value, tp ir.Type) (NumericalNode, error) {
	tp, err := concrete.Concrete(ev, tp)
	if err != nil {
		return nil, err
	}
	return ao.bck.ToElement(val, tp)
}

// Einsum calls an einstein sum on x and y given the expression in ref.
func (ao *Ops) Einsum(ev ir.Evaluator, ref *ir.EinsumExpr, x, y engine.NumericalElement) (engine.NumericalElement, error) {
	xNode, err := materialise.Element(ao, x)
	if err != nil {
		return nil, err
	}
	yNode, err := materialise.Element(ao, y)
	if err != nil {
		return nil, err
	}
	dotNode, err := ao.graph.DotGeneral(xNode, ref.ReduceAxes[0], ref.BatchAxes[0], yNode, ref.ReduceAxes[1], ref.BatchAxes[1], compute.DotGeneralConfig{})
	if err != nil {
		return nil, err
	}
	return ao.toElement(ev, dotNode, ref.Typ)
}

func elementsToInt(els []engine.NumericalElement) ([]int, error) {
	axes := make([]int, len(els))
	for i, el := range els {
		var err error
		axes[i], err = elements.IntFromElement(el)
		if err != nil {
			return nil, err
		}
	}
	return axes, nil
}

// BroadcastInDim the data of an array across dimensions.
func (ao *Ops) BroadcastInDim(ctx ir.Evaluator, expr ir.Expr, x engine.NumericalElement, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	axes, err := elementsToInt(axisLengths)
	if err != nil {
		return nil, err
	}
	xNode, xShape, err := materialise.ElementWithShape(ao, x)
	if err != nil {
		return nil, err
	}
	broadcastAxes := make([]int, 0)
	for i, xAxis := range xShape.Dimensions {
		if xAxis > 1 {
			continue
		}
		broadcastAxes = append(broadcastAxes, i)
	}
	targetShape := shapes.Shape{
		DType:      xShape.DType,
		Dimensions: axes,
	}
	reshaped, err := ao.graph.BroadcastInDim(xNode, targetShape, broadcastAxes)
	if err != nil {
		return nil, err
	}
	return ao.toElement(ctx, reshaped, expr.Type())
}

// Concat concatenates elements into a 1D array.
func (ao *Ops) Concat(ctx ir.Evaluator, expr ir.Expr, xs []engine.NumericalElement) (engine.NumericalElement, error) {
	nodes := make([]compute.Value, len(xs))
	for i, x := range xs {
		iNode, iShape, err := materialise.ElementWithShape(ao, x)
		if err != nil {
			return nil, err
		}
		size := iShape.Size()
		// Reshape elements to 1D arrays to work with Concat.
		iNodeArray, err := ao.graph.Reshape(iNode, size)
		if err != nil {
			return nil, err
		}
		nodes[i] = iNodeArray
	}
	array1d, err := ao.graph.Concatenate(0, nodes...)
	if err != nil {
		return nil, err
	}
	return ao.toElement(ctx, array1d, expr.Type())
}

// Set a slice in an array.
func (ao *Ops) Set(ctx ir.Evaluator, expr *ir.FuncCallExpr, x, updates ir.Element, position []ir.Element) (ir.Element, error) {
	xNode, xShape, err := materialise.ElementWithShape(ao, x)
	if err != nil {
		return nil, err
	}
	upNode, _, err := materialise.ElementWithShape(ao, updates)
	if err != nil {
		return nil, err
	}
	positionNodes, posShape, err := materialise.AllWithShapes(ao, position)
	if err != nil {
		return nil, err
	}
	rank := len(xShape.Dimensions)
	startIndices := make([]compute.Value, rank)
	copy(startIndices, positionNodes)
	if len(positionNodes) < rank {
		zeroShape := shapes.Make(posShape[0].DType)
		zeroNode, err := constantFromBytes(ao.graph, make([]byte, zeroShape.ByteSize()), zeroShape)
		if err != nil {
			return nil, err
		}
		for i := len(positionNodes); i < rank; i++ {
			startIndices[i] = zeroNode
		}
	}
	updateShape := make([]int, rank)
	for i := range rank {
		if i < len(position) {
			updateShape[i] = 1
		} else {
			updateShape[i] = xShape.Dimensions[i]
		}
	}
	upNodeReshaped, err := ao.graph.Reshape(upNode, updateShape...)
	if err != nil {
		return nil, err
	}
	setNode, err := ao.graph.DynamicUpdateSlice(xNode, upNodeReshaped, startIndices)
	if err != nil {
		return nil, err
	}
	return ao.toElement(ctx, setNode, expr.Type())
}

func constantFromBytes(g compute.Function, data []byte, sh shapes.Shape) (compute.Value, error) {
	if !sh.DType.IsSupported() || sh.DType.IsPacked() {
		return nil, errors.Errorf("cannot create constant: data type %v not supported", sh.DType)
	}
	if int64(len(data)) != sh.ByteSize() {
		return nil, errors.Errorf("buffer size is %d but shape specifies a buffer size of %d", len(data), sh.ByteSize())
	}
	flat := dtypes.UnsafeAnySliceFromBytes(unsafe.Pointer(unsafe.SliceData(data)), sh.DType, sh.Size())
	return g.Constant(flat, sh.Dimensions...)
}

func (ao *Ops) nodeFromArray(val *hostio.HostArray, typ ir.Type) (NumericalNode, error) {
	cstNode, err := ao.graph.Constant(val.FlatData(), val.Shape().Dimensions...)
	if err != nil {
		return nil, err
	}
	return ao.bck.ToElement(cstNode, typ)
}

// NodeFromArray returns a node from an array GX value.
func (ao *Ops) NodeFromArray(val *hostio.HostArray) (materialise.Node, error) {
	return ao.nodeFromArray(val, val.Type())
}

// ElementsFromNodes returns a slice of elements from nodes
func (ao *Ops) ElementsFromNodes(ev ir.Evaluator, nodes []compute.Value, types []ir.Type) ([]ir.Element, error) {
	els := make([]ir.Element, len(nodes))
	for i, node := range nodes {
		var err error
		els[i], err = ao.toElement(ev, node, types[i])
		if err != nil {
			return nil, err
		}
	}
	return els, nil
}

type constant struct {
	NumericalNode
	cst engine.Constant
}

var _ togo.CompileValue = (*constant)(nil)
var _ materialise.ElementMaterialiser = (*constant)(nil)

// GoValue of the underlying element.
func (c *constant) GoValue() (any, error) {
	return c.NumericalNode, nil
}

func (c *constant) Constant() engine.Constant {
	return c.cst
}

// Materialise returns the element with all its values from the graph.
func (c *constant) Materialise(materialise.Materialiser) (materialise.Node, error) {
	return c, nil
}

// ElementFromHostValue returns transforms an atomic literal element into an element specific to the ArrayOps implementation.
func (ao *Ops) ElementFromHostValue(ev ir.Evaluator, cst engine.Constant) (engine.ConstantElement, error) {
	ctyp, err := concrete.Concrete(ev, cst.Type())
	if err != nil {
		return nil, err
	}
	cArray := ir.ToArrayType(ctyp)
	if cArray == nil {
		return nil, fmterr.Internalf("cannot get array type from %s", ctyp.ReferString(ev.File()))
	}
	// The constant is built given the kind of its data type,
	// which is different from the kind of the type for arrays.
	val, err := cst.BuildNode(ao.graph, cArray.DataType().Kind())
	if err != nil {
		return nil, err
	}
	el, err := ao.bck.ToElement(val, ctyp)
	if err != nil {
		return nil, err
	}
	return &constant{NumericalNode: el, cst: cst}, nil
}

// DefineGlobalConst defines a global constant for the interpreter.
func (ao *Ops) DefineGlobalConst(_ ir.Storage, el ir.Element) (ir.Element, error) {
	return el, nil
}

// EvalCond evaluates a condition, typically for an if statement.
func (ao *Ops) EvalCond(ev ir.Evaluator, el ir.Element) (bool, error) {
	return constants.Convert(constants.CBool, el)
}

func (ao *Ops) axesFromShape(ctx ir.Evaluator, axlens []int) (*elements.Slice, error) {
	axes := make([]ir.Element, len(axlens))
	for i, axisSize := range axlens {
		iEl, err := numbers.NewConstant(ir.IntType(), axisSize)
		if err != nil {
			return nil, err
		}
		axes[i], err = ao.ElementFromHostValue(ctx, iEl)
		if err != nil {
			return nil, err
		}
	}
	return elements.NewSlice(ir.IntSliceType(), axes)
}

func (ao *Ops) elementFromValue(ev ir.Evaluator, val hostio.Value) (ir.Element, error) {
	if devArray, ok := val.(*hostio.DeviceArray); ok {
		return ao.toElement(ev, devArray.Buffer(), val.Type())
	}
	return val, nil
}

// ElementFromReceiverValue converts a function receiver value into an element.
func (ao *Ops) ElementFromReceiverValue(ev ir.Evaluator, _ *ir.Field, val hostio.Value) (ir.Element, error) {
	return ao.elementFromValue(ev, val)
}

// ElementFromArgValue converts a function argument value into an element.
func (ao *Ops) ElementFromArgValue(ev ir.Evaluator, _ *ir.Field, _ int, val hostio.Value) (ir.Element, error) {
	return ao.elementFromValue(ev, val)
}

// Values returns the GX values computed by the backend for elements.
// Slice elements are flattened.
func (ao *Ops) Values(els []ir.Element) ([]hostio.Value, error) {
	vals := make([]hostio.Value, len(els))
	for i, el := range els {
		var err error
		if vals[i], err = cast.To[hostio.Value](el); err != nil {
			return nil, err
		}
	}
	vals, err := elements.Flatten(vals)
	if err != nil {
		return nil, err
	}
	for i, val := range vals {
		if _, ok := val.(engine.ConstantElement); ok {
			if vals[i], err = togo.CpValueT[hostio.Value](val); err != nil {
				return nil, err
			}
		}
	}
	return vals, nil
}
