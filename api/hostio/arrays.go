// Copyright 2026 Google LLC
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

package hostio

import (
	"fmt"
	"math/big"
	"strings"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gomlx/compute/dtypes/bfloat16"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/dtypes/gotype"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/fmt/fmtarray"
	"github.com/gx-org/gx/internal/togo"
)

// Since uintptr cannot be negative, each of these statements will trigger a compile-time error iff
// the first type is smaller than the second type.
const (
	// Ensure int64 and Go int are the same size.
	_ = unsafe.Sizeof(int64(0)) - unsafe.Sizeof(int(0))
	_ = unsafe.Sizeof(int(0)) - unsafe.Sizeof(int64(0))
)

// Array is an array value (also includes atomic).
type Array interface {
	Value
	togo.CompileValue

	// Type returns the GX type of the array.
	Type() ir.Type

	// Shape of the array.
	Shape() shapes.Shape

	// ToDevice transfers the array to a device.
	// It is a no-op if the data is already on the device.
	ToDevice(bck compute.Backend, dev compute.DeviceNum) (*DeviceArray, error)

	// ToHostArray transfers the array on the host if it is not already.
	ToHostArray() (*HostArray, error)
}

type baseArray struct {
	typ       ir.Type
	arrayType ir.ArrayType
}

func newBaseArray(typ ir.Type, sh shapes.Shape) (*baseArray, error) {
	arrayType, err := ir.ToArrayTypeGivenShape(nil, typ, sh)
	if err != nil {
		return nil, err
	}
	return &baseArray{typ: typ, arrayType: arrayType}, nil
}

// DeviceArray managed by GX where the data is on a device.
type DeviceArray struct {
	*baseArray
	handle compute.Buffer
}

var _ Array = (*DeviceArray)(nil)

// NewDeviceArray returns a new array managed by GX.
func NewDeviceArray(typ ir.Type, handle compute.Buffer) (*DeviceArray, error) {
	sh, err := handle.Shape()
	if err != nil {
		return nil, err
	}
	base, err := newBaseArray(typ, sh)
	if err != nil {
		return nil, err
	}
	return &DeviceArray{
		baseArray: base,
		handle:    handle,
	}, nil
}

// ToHost transfers the value to the host.
func (a *DeviceArray) ToHost() (Value, error) {
	return a.ToHostArray()
}

// Finalize releases the device buffer.
func (a *DeviceArray) Finalize() error {
	return a.handle.Finalize()
}

// GoValue of the underlying element.
func (a *DeviceArray) GoValue() (any, error) {
	hVal, err := a.ToHostArray()
	if err != nil {
		return nil, err
	}
	return hVal.GoValue()
}

// ToHostArray transfers the array to the host.
func (a *DeviceArray) ToHostArray() (*HostArray, error) {
	sh := a.Shape()
	data, err := allocateHostData(sh)
	if err != nil {
		return nil, err
	}
	if err := a.handle.ToFlatData(data.flatData()); err != nil {
		return nil, err
	}
	base, err := newBaseArray(a.typ, sh)
	if err != nil {
		return nil, err
	}
	return &HostArray{baseArray: base, data: data}, nil
}

// Type of the array.
func (a *DeviceArray) Type() ir.Type {
	return a.typ
}

// Shape of the array.
func (a *DeviceArray) Shape() shapes.Shape {
	sh, err := a.handle.Shape()
	if err != nil {
		panic(err)
	}
	return sh
}

// Buffer returns the buffer pointing to the data on the device.
func (a *DeviceArray) Buffer() compute.Buffer {
	return a.handle
}

// ToDevice transfers the data to a device.
// It is a no-op if the data is already on the device.
func (a *DeviceArray) ToDevice(bck compute.Backend, dev compute.DeviceNum) (*DeviceArray, error) {
	devNum, err := a.handle.DeviceNum()
	if err != nil {
		return nil, err
	}
	if a.handle.Backend() == bck {
		if devNum == dev {
			return a, nil
		}
		handle, err := a.handle.CopyToDevice(dev)
		if err != nil {
			return nil, err
		}
		return NewDeviceArray(a.typ, handle)
	}
	host, err := a.ToHostArray()
	if err != nil {
		return nil, err
	}
	return host.ToDevice(bck, dev)
}

// SourceString returns the GX source code of the implementation.
func (a *DeviceArray) SourceString(from *ir.File) string {
	host, err := a.ToHost()
	var hostS string
	if err != nil {
		hostS = err.Error()
	} else {
		hostS = ir.DefineString(from, host)
	}
	devNum, _ := a.handle.DeviceNum()
	return fmt.Sprintf("DeviceArray{DeviceID: %d}: %s", devNum, hostS)
}

// HostData stores the data of an array on the host.
type HostData[T gotype.Supported] struct {
	Shape shapes.Shape
	Flat  []T
}

type hostData interface {
	shape() shapes.Shape
	flatData() any
	bytes() []byte
}

func (d *HostData[T]) shape() shapes.Shape { return d.Shape }
func (d *HostData[T]) flatData() any       { return d.Flat }
func (d *HostData[T]) bytes() []byte       { return dtypes.UnsafeByteSlice(d.Flat) }

func allocateHostData(sh shapes.Shape) (hostData, error) {
	switch sh.DType {
	case dtypes.Bool:
		return &HostData[bool]{Shape: sh, Flat: make([]bool, sh.Size())}, nil
	case dtypes.BFloat16:
		return &HostData[bfloat16.BFloat16]{Shape: sh, Flat: make([]bfloat16.BFloat16, sh.Size())}, nil
	case dtypes.Float32:
		return &HostData[float32]{Shape: sh, Flat: make([]float32, sh.Size())}, nil
	case dtypes.Float64:
		return &HostData[float64]{Shape: sh, Flat: make([]float64, sh.Size())}, nil
	case dtypes.Int32:
		return &HostData[int32]{Shape: sh, Flat: make([]int32, sh.Size())}, nil
	case dtypes.Int64:
		return &HostData[int64]{Shape: sh, Flat: make([]int64, sh.Size())}, nil
	case dtypes.Uint32:
		return &HostData[uint32]{Shape: sh, Flat: make([]uint32, sh.Size())}, nil
	case dtypes.Uint64:
		return &HostData[uint64]{Shape: sh, Flat: make([]uint64, sh.Size())}, nil
	default:
		return nil, errors.Errorf("cannot create an array of data type %s: not supported", sh.DType)
	}
}

// HostArray managed by GX where the data is on the host.
type HostArray struct {
	*baseArray
	data hostData
}

var _ Array = (*HostArray)(nil)

// NewHostArrayFromData returns a new host array from a HostData structure.
func NewHostArrayFromData[T gotype.Supported](typ ir.Type, data *HostData[T]) (*HostArray, error) {
	if data == nil {
		return nil, errors.Errorf("host data is nil")
	}
	sh := data.Shape
	switch sh.DType {
	case dtypes.Bool, dtypes.BFloat16, dtypes.Float32, dtypes.Float64, dtypes.Int32, dtypes.Int64, dtypes.Uint32, dtypes.Uint64:
	default:
		return nil, errors.Errorf("cannot create an array of data type %s: not supported", sh.DType)
	}
	wantDType := dtypes.FromGenericsType[T]()
	if sh.DType != wantDType {
		return nil, errors.Errorf("host data shape DType is %s, want %s for %T", sh.DType, wantDType, *new(T))
	}
	if len(data.Flat) != sh.Size() {
		return nil, errors.Errorf("flat slice length is %d but shape %s specifies %d elements", len(data.Flat), sh, sh.Size())
	}
	base, err := newBaseArray(typ, sh)
	if err != nil {
		return nil, err
	}
	var hData hostData = data
	if intFlat, ok := any(data.Flat).([]int); ok {
		var int64Flat []int64
		if len(intFlat) > 0 {
			int64Flat = unsafe.Slice((*int64)(unsafe.Pointer(unsafe.SliceData(intFlat))), len(intFlat))
		}
		hData = &HostData[int64]{Shape: sh, Flat: int64Flat}
	}
	return &HostArray{baseArray: base, data: hData}, nil
}

// NewHostArrayFromFlat returns a new host array from a flat Go slice and axis lengths.
func NewHostArrayFromFlat[T gotype.Supported](typ ir.Type, flat []T, dims ...int) (*HostArray, error) {
	sh := shapes.Make(dtypes.FromGenericsType[T](), dims...)
	return NewHostArrayFromData(typ, &HostData[T]{Shape: sh, Flat: flat})
}

// Type of the array.
func (a *HostArray) Type() ir.Type {
	return a.typ
}

// Shape of the array.
func (a *HostArray) Shape() shapes.Shape {
	return a.data.shape()
}

// FlatData returns the untyped slice storing the array data.
func (a *HostArray) FlatData() any {
	return a.data.flatData()
}

// Bytes returns the raw byte slice of the array data.
func (a *HostArray) Bytes() []byte {
	return a.data.bytes()
}

// ToHost returns receiver.
func (a *HostArray) ToHost() (Value, error) {
	return a, nil
}

// Finalize is a no-op for arrays on the host.
func (a *HostArray) Finalize() error {
	return nil
}

// ToHostArray returns the receiver.
func (a *HostArray) ToHostArray() (*HostArray, error) {
	return a, nil
}

// ToDevice transfers the data to a device.
// It is a no-op if the data is already on the device.
func (a *HostArray) ToDevice(bck compute.Backend, dev compute.DeviceNum) (*DeviceArray, error) {
	handle, err := bck.BufferFromFlatData(dev, a.FlatData(), a.Shape())
	if err != nil {
		return nil, err
	}
	return NewDeviceArray(a.typ, handle)
}

// Flat returns the typed slice of a host array.
func Flat[T gotype.Supported](a *HostArray) ([]T, error) {
	var zero T
	sh := a.Shape()
	if wantDType := dtypes.FromGenericsType[T](); sh.DType != wantDType {
		return nil, errors.Errorf("array data type is %s, want %s", sh.DType, wantDType)
	}
	if d, ok := a.data.(*HostData[T]); ok {
		return d.Flat, nil
	}
	if _, isInt := any(zero).(int); isInt {
		if d64, ok := a.data.(*HostData[int64]); ok {
			if len(d64.Flat) == 0 {
				return nil, nil
			}
			s := unsafe.Slice((*int)(unsafe.Pointer(unsafe.SliceData(d64.Flat))), len(d64.Flat))
			return any(s).([]T), nil
		}
	}
	return nil, errors.Errorf("unexpected host data type %T for %T", a.data, zero)
}

func atomFromFlat(flat any) (any, error) {
	switch flat := flat.(type) {
	case []bool:
		return flat[0], nil
	case []bfloat16.BFloat16:
		return flat[0], nil
	case []float32:
		return flat[0], nil
	case []float64:
		return flat[0], nil
	case []int32:
		return flat[0], nil
	case []int64:
		return flat[0], nil
	case []uint32:
		return flat[0], nil
	case []uint64:
		return flat[0], nil
	default:
		return nil, errors.Errorf("cannot get atom from slice of type %T", flat)
	}
}

// ToAtom returns the value as an atomic value.
// An error is returned if the array contains more than one value.
func (a *HostArray) ToAtom() (any, error) {
	sh := a.Shape()
	if !sh.IsScalar() {
		return nil, errors.Errorf("%s not atomic", sh.String())
	}
	return atomFromFlat(a.FlatData())
}

type goArray struct {
	flat any
	dims []int
}

func (a goArray) String() string {
	switch flat := a.flat.(type) {
	case []bool:
		return fmtarray.Sprint(flat, a.dims)
	case []bfloat16.BFloat16:
		return fmtarray.Sprint(flat, a.dims)
	case []float32:
		return fmtarray.Sprint(flat, a.dims)
	case []float64:
		return fmtarray.Sprint(flat, a.dims)
	case []int32:
		return fmtarray.Sprint(flat, a.dims)
	case []int64:
		return fmtarray.Sprint(flat, a.dims)
	case []uint32:
		return fmtarray.Sprint(flat, a.dims)
	case []uint64:
		return fmtarray.Sprint(flat, a.dims)
	default:
		return fmt.Sprintf("unsupported slice type %T", a.flat)
	}
}

// GoValue of the underlying element.
func (a *HostArray) GoValue() (any, error) {
	flat := a.FlatData()
	sh := a.Shape()
	if sh.IsScalar() {
		return atomFromFlat(flat)
	}
	return goArray{flat: flat, dims: sh.Dimensions}, nil
}

// ToFloatNumber returns the value as a float number.
// An error is returned if the array contains more than one value.
func (a *HostArray) ToFloatNumber() (*big.Float, error) {
	atom, err := a.ToAtom()
	if err != nil {
		return nil, err
	}
	switch v := atom.(type) {
	case bfloat16.BFloat16:
		return big.NewFloat(float64(v.Float32())), nil
	case float32:
		return big.NewFloat(float64(v)), nil
	case float64:
		return big.NewFloat(v), nil
	case int32:
		return big.NewFloat(float64(v)), nil
	case int64:
		return new(big.Float).SetInt64(v), nil
	case uint32:
		return big.NewFloat(float64(v)), nil
	case uint64:
		return new(big.Float).SetUint64(v), nil
	default:
		return nil, errors.Errorf("cannot convert non-algebra type %s to big.Float", a.Shape().DType)
	}
}

// SourceString returns the values of the array as a string without the type.
func (a *HostArray) SourceString(from *ir.File) string {
	return a.sourceString(from, false)
}

// DefineString returns the GX source code of the implementation.
func (a *HostArray) DefineString(from *ir.File) string {
	return a.sourceString(from, true)
}

func sDataPrint(flat any, dims []int) string {
	switch flat := flat.(type) {
	case []bool:
		return fmtarray.SDataPrint(flat, dims)
	case []bfloat16.BFloat16:
		return fmtarray.SDataPrint(flat, dims)
	case []float32:
		return fmtarray.SDataPrint(flat, dims)
	case []float64:
		return fmtarray.SDataPrint(flat, dims)
	case []int32:
		return fmtarray.SDataPrint(flat, dims)
	case []int64:
		return fmtarray.SDataPrint(flat, dims)
	case []uint32:
		return fmtarray.SDataPrint(flat, dims)
	case []uint64:
		return fmtarray.SDataPrint(flat, dims)
	default:
		return fmt.Sprintf("unsupported slice type %T", flat)
	}
}

func (a *HostArray) sourceString(from *ir.File, withType bool) string {
	shape := a.Shape().Dimensions
	dataS := sDataPrint(a.FlatData(), shape)
	if !withType {
		return dataS
	}
	dims := make([]string, len(shape))
	for i, axLen := range shape {
		dims[i] = fmt.Sprintf("[%d]", axLen)
	}
	if len(shape) == 0 {
		dataS = fmt.Sprintf("(%s)", dataS)
	}
	namedType, isNamed := a.typ.(*ir.NamedType)
	if isNamed {
		return fmt.Sprintf("%s%s", namedType.ReferString(from), dataS)
	}
	return fmt.Sprintf("%s%s%s", strings.Join(dims, ""), a.arrayType.DataType().ReferString(from), dataS)
}

// String representation of the array and its type.
func (a *HostArray) String() string {
	return a.DefineString(nil)
}

// ToAtom converts an array on the host into a Go atom value.
func ToAtom[T gotype.Supported](a *HostArray) (T, error) {
	var zero T
	sh := a.Shape()
	if !sh.IsScalar() {
		return zero, errors.Errorf("array %s (length=%d) is not an atom", sh.String(), sh.Size())
	}
	slice, err := Flat[T](a)
	if err != nil {
		return zero, err
	}
	if len(slice) != 1 {
		return zero, errors.Errorf("array (length=%d) is not an atom", len(slice))
	}
	return slice[0], nil
}
