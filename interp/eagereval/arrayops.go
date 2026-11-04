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

package eagereval

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gomlx/compute/dtypes/bfloat16"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/concrete"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/internal/interp/constants"
	"github.com/gx-org/gx/internal/interp/flatten"
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
)

func newRegistry(bck EagerBackend, importer ir.Importer, fn compute.Function) (*engine.Registry, *Ops) {
	ao := &Ops{bck: bck, graph: fn}
	proc := processor.New()
	return &engine.Registry{
		Importer:       importer,
		CompEvalEngine: compeval.Engine(),
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
	if int64(len(data)) != sh.ByteSize() {
		return nil, errors.Errorf("buffer size is %d but shape specifies a buffer size of %d", len(data), sh.ByteSize())
	}
	var flat any
	switch sh.DType {
	case dtypes.Bool:
		flat = kernels.ToSlice[bool](data)
	case dtypes.BFloat16:
		flat = kernels.ToSlice[bfloat16.BFloat16](data)
	case dtypes.Float32:
		flat = kernels.ToSlice[float32](data)
	case dtypes.Float64:
		flat = kernels.ToSlice[float64](data)
	case dtypes.Int32:
		flat = kernels.ToSlice[int32](data)
	case dtypes.Int64:
		flat = kernels.ToSlice[int64](data)
	case dtypes.Uint32:
		flat = kernels.ToSlice[uint32](data)
	case dtypes.Uint64:
		flat = kernels.ToSlice[uint64](data)
	default:
		return nil, errors.Errorf("cannot create constant: data type %v not supported", sh.DType)
	}
	return g.Constant(flat, sh.Dimensions...)
}

func (ao *Ops) nodeFromArray(val *hostio.HostArray, typ ir.Type) (NumericalNode, error) {
	buf := val.Buffer()
	data := buf.Acquire()
	defer buf.Release()
	cstNode, err := constantFromBytes(ao.graph, data, val.Shape())
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

func checkMissingParams(fn ir.Func, got []hostio.Value, want []*ir.Field) error {
	if len(got) == len(want) {
		return nil
	}
	if len(got) > len(want) {
		return errors.Errorf("too many arguments in call to %s: have %d but want %d", fn.FuncType().ReferString(nil), len(got), len(want))
	}
	missingParams := want[len(got):]
	builder := strings.Builder{}
	for n, param := range missingParams {
		if n > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(param.Name.String())
	}
	return errors.Errorf("missing parameter(s): %s", builder.String())
}

func (ao *Ops) convertScalarToElement(ev ir.Evaluator, typ ir.Type, val kernels.Array) (ir.Element, error) {
	atom, err := val.ToAtom()
	if err != nil {
		return nil, err
	}
	cst, err := numbers.NewConstant(typ, atom)
	if err != nil {
		return nil, err
	}
	return ao.ElementFromHostValue(ev, cst)
}

func (ao *Ops) convertSliceToElement(ev ir.Evaluator, field *ir.Field, typ ir.Type, val kernels.Array) (ir.Element, error) {
	sliceType, isSlice := typ.(*ir.SliceType)
	if !isSlice {
		return nil, errors.Errorf("cannot convert array argument to %s for parameter %s", typ.ReferString(nil), field.Name.Name)
	}
	subType, ok := sliceType.ElementType()
	if !ok {
		return nil, errors.Errorf("cannot get element type of %s for parameter %s", typ.ReferString(nil), field.Name.Name)
	}
	sh := val.Shape()
	numEls := sh.Dimensions[0]
	els := make([]ir.Element, numEls)
	for i := range numEls {
		sliceKernel, _, err := val.Factory().Slice(sh, i)
		if err != nil {
			return nil, err
		}
		subVal, err := sliceKernel(val)
		if err != nil {
			return nil, err
		}
		els[i], err = ao.convertKernelArrayToElement(ev, field, subType, subVal)
		if err != nil {
			return nil, err
		}
	}
	return elements.NewSlice(typ, els)
}

func (ao *Ops) convertKernelArrayToElement(ev ir.Evaluator, field *ir.Field, typ ir.Type, val kernels.Array) (ir.Element, error) {
	if val.Shape().IsScalar() {
		return ao.convertScalarToElement(ev, typ, val)
	}
	return ao.convertSliceToElement(ev, field, typ, val)
}

func (ao *Ops) convertToElement(ev ir.Evaluator, field *ir.Field, arg ir.Element) (ir.Element, error) {
	array, isArray := arg.(hostio.Array)
	if !isArray {
		return arg, nil
	}
	val, err := togo.CpValueT[kernels.Array](array)
	if err != nil {
		return nil, err
	}
	return ao.convertKernelArrayToElement(ev, field, field.Type(), val)
}

// funcInputsToElements converts the receiver and the arguments of a function call to elements.
func (ao *Ops) funcInputsToElements(ev ir.Evaluator, fn ir.Func, receiver hostio.Value, args []hostio.Value) (*elements.InputElements, error) {
	fType := fn.FuncType()
	params := fType.AllParams()
	if err := checkMissingParams(fn, args, params); err != nil {
		return nil, err
	}
	in := &elements.InputElements{
		Receiver: receiver,
		Args:     make([]ir.Element, len(args)),
	}
	for i, arg := range args {
		if devArray, ok := arg.(*hostio.DeviceArray); ok {
			var err error
			arg, err = ao.toElement(ev, devArray.Buffer(), arg.Type())
			if err != nil {
				return nil, err
			}
		}
		in.Args[i] = arg
	}
	return in, nil
}

// Values returns the GX values computed by the backend for elements.
// Composite elements are flattened.
func (ao *Ops) Values(els []ir.Element) ([]hostio.Value, error) {
	els, err := flatten.Flatten(els...)
	if err != nil {
		return nil, err
	}
	nodes, err := materialise.All(ao, els)
	if err != nil {
		return nil, err
	}
	vals := make([]hostio.Value, len(nodes))
	for i, node := range nodes {
		if vals[i], err = cast.To[hostio.Value](node); err != nil {
			return nil, err
		}
	}
	return vals, nil
}
