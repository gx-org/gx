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

package flatten

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/api/values"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/ir"
)

// Parser unflattens the output of a graph computation
// into GX values.
type Parser struct {
	dev        platform.Device
	callInputs *values.FuncInputs
	// Handles to unflatten.
	compOutput []platform.DeviceHandle
	nextPos    int
}

// NewParser returns a new parser given the output of a graph computation.
func NewParser(dev platform.Device, callInputs *values.FuncInputs, handles []platform.DeviceHandle) *Parser {
	return &Parser{
		dev:        dev,
		callInputs: callInputs,
		compOutput: handles,
	}
}

// Next returns a the next handle and moves the cursor to the succeeding handle.
func (h *Parser) Next() platform.DeviceHandle {
	n := h.compOutput[h.nextPos]
	h.nextPos++
	return n
}

// CallInputs returns the inputs with which the function was called.
func (h *Parser) CallInputs() *values.FuncInputs {
	return h.callInputs
}

func (h *Parser) size() int {
	return len(h.compOutput)
}

// Device returns to which transfers the host value to.
// TODO(b/388207169): Always transfer the value to device because C++ bindings do not support HostValue.
func (h *Parser) Device() platform.Device {
	return h.dev
}

// Unflatten consumes the next available handles and returns a GX value matching the given element.
func (h *Parser) Unflatten(el ir.Element) (values.Value, error) {
	unf, ok := el.(Unflattener)
	if !ok {
		return nil, errors.Errorf("cannot unflatten element type %T", el)
	}
	val, err := unf.Unflatten(h)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, errors.Errorf("state element %T returned a nil value", el)
	}
	return val, nil
}

type newCompValue func(ir.Type, []values.Value) (values.Value, error)

// ParseArray the next value as an array.
func (h *Parser) ParseArray(typ ir.Type) (values.Array, error) {
	return values.NewDeviceArray(typ, h.Next())
}

// ParseComposite unflatten a slice of elements into a single GX value.
func (h *Parser) ParseComposite(ncv newCompValue, typ ir.Type, els []ir.Element) (values.Value, error) {
	vals := make([]values.Value, len(els))
	for i, el := range els {
		var err error
		vals[i], err = h.Unflatten(el)
		if err != nil {
			return nil, err
		}
	}
	return ncv(typ, vals)
}

func (h *Parser) String() string {
	var s strings.Builder
	fmt.Fprintf(&s, "%T{", h)
	for i, hdl := range h.compOutput {
		prefix := "  "
		if i == h.nextPos {
			prefix = "->"
		}
		fmt.Fprintf(&s, "%s%d: %s\n", prefix, i, gxfmt.String(hdl))
	}
	s.WriteString("}")
	return s.String()
}

// ParseCompositeOf returns a function to unflatten a composite value.
func ParseCompositeOf[T values.Value](
	f func(ir.Type, []values.Value) (T, error),
) func(ir.Type, []values.Value) (values.Value, error) {
	return func(typ ir.Type, vals []values.Value) (values.Value, error) {
		return f(typ, vals)
	}
}
