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

// Package parameters_go_gx are Go bindings to the GX package:
// github.com/gx-org/gx/tests/bindings/parameters.
//
// Automatically generated by
// gx/golang/binder/gobindings/bindings.go.tmpl.
package parameters_go_gx

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/trace"
	"github.com/gx-org/gx/api/tracer"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
	_ "github.com/gx-org/gx/tests/bindings/parameters"
	"github.com/pkg/errors"
)

// Force some package dependencies.
var (
	_ = fmt.Println
	_ = strings.Compare
	_ = reflect.TypeFor[int]
	_ = values.Struct{}
	_ = errors.Errorf
	_ = types.NewSlice[types.Bridger]
	_ = platform.HostTransfer
)

// PackageIR is the GX package intermediate representation
// built for a given runtime, but not yet for a specific device.
type PackageIR struct {
	Runtime *api.Runtime
	IR      *ir.Package
	Tracer  trace.Callback
}

// Load the GX package for a given backend.
func Load(rtm *api.Runtime) (*PackageIR, error) {
	bpkg, err := rtm.Builder().Build("github.com/gx-org/gx/tests/bindings/parameters")
	if err != nil {
		return nil, err
	}
	pkg := &PackageIR{
		Runtime: rtm,
		IR:      bpkg.IR(),
	}

	return pkg, nil
}

// BuildFor loads the GX package github.com/gx-org/gx/tests/bindings/parameters
// then returns that package for a given device and options.
func BuildFor(dev *api.Device, options ...options.PackageOptionFactory) (*Package, error) {
	pkg, err := Load(dev.Runtime())
	if err != nil {
		return nil, err
	}
	return pkg.BuildFor(dev, options...), nil
}

// Factory create new instance of types used in the package.
// The compiler associated with the factory defines on what
// device and with which options methods of the instances
// created by the factory are compiled for.
type Factory struct {
	Package *Package
}

// Package is a GX package for a given device.
// Functions and methods are compiled specifically for that device.
type Package struct {
	Package *PackageIR
	Device  *api.Device
	Factory *Factory

	options []options.PackageOption

	NewStruct                 NewStruct
	AddToStruct               AddToStruct
	AddFloat32                AddFloat32
	AddInt                    AddInt
	AddFloat32s               AddFloat32s
	AddInts                   AddInts
	Len                       Len
	Iota                      Iota
	SliceArrayArgConstIndex   SliceArrayArgConstIndex
	SliceArrayArg             SliceArrayArg
	SliceSliceArg             SliceSliceArg
	NewNotInSlice             NewNotInSlice
	methodStructSetNotInSlice methodBase
}

// AppendOptions appends options to the compiler.
func (cmpl *Package) AppendOptions(options ...options.PackageOptionFactory) {
	plat := cmpl.Package.Runtime.Backend().Platform()
	for _, opt := range options {
		cmpl.options = append(cmpl.options, opt(plat))
	}
}

// BuildFor returns a package ready to compile for a device and options.
func (pkg *PackageIR) BuildFor(dev *api.Device, options ...options.PackageOptionFactory) *Package {
	c := &Package{
		Package: pkg,
		Device:  dev,
	}
	c.Factory = &Factory{Package: c}
	c.AppendOptions(options...)

	c.NewStruct = NewStruct{
		methodBase: methodBase{
			pkg:      c,
			function: c.Package.IR.Decls.Funcs[0],
		},
	}
	c.AddToStruct = AddToStruct{
		methodBase: methodBase{
			pkg:      c,
			function: c.Package.IR.Decls.Funcs[3],
		},
	}
	c.AddFloat32 = AddFloat32{
		methodBase: methodBase{
			pkg:      c,
			function: c.Package.IR.Decls.Funcs[4],
		},
	}
	c.AddInt = AddInt{
		methodBase: methodBase{
			pkg:      c,
			function: c.Package.IR.Decls.Funcs[5],
		},
	}
	c.AddFloat32s = AddFloat32s{
		methodBase: methodBase{
			pkg:      c,
			function: c.Package.IR.Decls.Funcs[6],
		},
	}
	c.AddInts = AddInts{
		methodBase: methodBase{
			pkg:      c,
			function: c.Package.IR.Decls.Funcs[7],
		},
	}
	c.Len = Len{
		methodBase: methodBase{
			pkg:      c,
			function: c.Package.IR.Decls.Funcs[8],
		},
	}
	c.Iota = Iota{
		methodBase: methodBase{
			pkg:      c,
			function: c.Package.IR.Decls.Funcs[9],
		},
	}
	c.SliceArrayArgConstIndex = SliceArrayArgConstIndex{
		methodBase: methodBase{
			pkg:      c,
			function: c.Package.IR.Decls.Funcs[10],
		},
	}
	c.SliceArrayArg = SliceArrayArg{
		methodBase: methodBase{
			pkg:      c,
			function: c.Package.IR.Decls.Funcs[11],
		},
	}
	c.SliceSliceArg = SliceSliceArg{
		methodBase: methodBase{
			pkg:      c,
			function: c.Package.IR.Decls.Funcs[12],
		},
	}
	c.NewNotInSlice = NewNotInSlice{
		methodBase: methodBase{
			pkg:      c,
			function: c.Package.IR.Decls.Funcs[13],
		},
	}

	c.methodStructSetNotInSlice = methodBase{
		pkg:      c,
		function: c.Package.IR.Decls.Types[2].Methods[0],
	}

	return c
}

var Size SizeStatic

type SizeStatic struct {
	value ir.Int
}

func (SizeStatic) Set(value ir.Int) options.PackageOptionFactory {
	return func(plat platform.Platform) options.PackageOption {
		hostValue := types.DefaultInt(value)
		return options.PackageVarSetValue{
			Pkg:   "github.com/gx-org/gx/tests/bindings/parameters",
			Var:   "Size",
			Value: hostValue.GXValue(),
		}
	}
}

// handleNotInSlice stores the backend handles of NotInSlice.
type handleNotInSlice struct {
	pkg   *Package
	struc *ir.NamedType
	owner *NotInSlice
}

// Type of the value.
func (h *handleNotInSlice) Type() ir.Type {
	return h.struc
}

// NamedType returns the intermediate representation of the type.
func (h *handleNotInSlice) NamedType() *ir.NamedType {
	return h.struc
}

// Bridger returns the Go object owning this handle.
func (h *handleNotInSlice) Bridger() types.Bridger {
	return h.owner
}

// GXValue returns the GX value.
func (h *handleNotInSlice) GXValue() values.Value {
	return h.owner.value
}

// String representation of the handle.
func (h *handleNotInSlice) String() string {
	bld := strings.Builder{}
	bld.WriteString("NotInSlice{\n")

	bld.WriteString(fmt.Sprintf("%s:%s\n", "Val", any(h.owner.Val).(fmt.Stringer).String()))

	bld.WriteString("}")
	return bld.String()
}

// NotInSlice stores the handle of NotInSlice on a device.
type NotInSlice struct {
	handle handleNotInSlice
	value  *values.NamedType

	Val types.Atom[int32]
}

var (
	_ types.Bridger      = (*NotInSlice)(nil)
	_ types.StructBridge = (*handleNotInSlice)(nil)
)

// StructValue returns the GX value of the structure.
func (h *handleNotInSlice) StructValue() *values.Struct {
	return h.owner.value.Underlying().(*values.Struct)
}

// MarshalNotInSlice populates the receiver fields with device handles.
func (cmpl *Package) MarshalNotInSlice(val values.Value) (s *NotInSlice, err error) {
	s = cmpl.Factory.NewNotInSlice()
	var ok bool
	s.value, ok = val.(*values.NamedType)
	if !ok {
		err = errors.Errorf("cannot use handle to set NotInSlice: %T is not a %s", val, reflect.TypeFor[*values.NamedType]())
		return
	}
	structVal, ok := s.value.Underlying().(*values.Struct)
	if !ok {
		err = errors.Errorf("incorrect underlying value for named type NotInSlice: %T is not a %s", val, reflect.TypeFor[*values.Struct]().Name())
		return
	}
	fields := make([]values.Value, structVal.StructType().NumFields())
	for i, field := range structVal.StructType().Fields.Fields() {
		fields[i] = structVal.FieldValue(field.Name.Name)
	}

	field0Value, ok := fields[0].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", fields[0], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	field0 := types.NewAtom[int32](field0Value)

	s.Val = field0
	return
}

func (s NotInSlice) String() string {
	return s.handle.String()
}

// Bridge returns the bridge between the Go value and the GX value.
func (s *NotInSlice) Bridge() types.Bridge { return &s.handle }

// handleInSlice stores the backend handles of InSlice.
type handleInSlice struct {
	pkg   *Package
	struc *ir.NamedType
	owner *InSlice
}

// Type of the value.
func (h *handleInSlice) Type() ir.Type {
	return h.struc
}

// NamedType returns the intermediate representation of the type.
func (h *handleInSlice) NamedType() *ir.NamedType {
	return h.struc
}

// Bridger returns the Go object owning this handle.
func (h *handleInSlice) Bridger() types.Bridger {
	return h.owner
}

// GXValue returns the GX value.
func (h *handleInSlice) GXValue() values.Value {
	return h.owner.value
}

// String representation of the handle.
func (h *handleInSlice) String() string {
	bld := strings.Builder{}
	bld.WriteString("InSlice{\n")

	bld.WriteString(fmt.Sprintf("%s:%s\n", "Val", any(h.owner.Val).(fmt.Stringer).String()))

	bld.WriteString("}")
	return bld.String()
}

// InSlice stores the handle of InSlice on a device.
type InSlice struct {
	handle handleInSlice
	value  *values.NamedType

	Val types.Atom[int32]
}

var (
	_ types.Bridger      = (*InSlice)(nil)
	_ types.StructBridge = (*handleInSlice)(nil)
)

// StructValue returns the GX value of the structure.
func (h *handleInSlice) StructValue() *values.Struct {
	return h.owner.value.Underlying().(*values.Struct)
}

// MarshalInSlice populates the receiver fields with device handles.
func (cmpl *Package) MarshalInSlice(val values.Value) (s *InSlice, err error) {
	s = cmpl.Factory.NewInSlice()
	var ok bool
	s.value, ok = val.(*values.NamedType)
	if !ok {
		err = errors.Errorf("cannot use handle to set InSlice: %T is not a %s", val, reflect.TypeFor[*values.NamedType]())
		return
	}
	structVal, ok := s.value.Underlying().(*values.Struct)
	if !ok {
		err = errors.Errorf("incorrect underlying value for named type InSlice: %T is not a %s", val, reflect.TypeFor[*values.Struct]().Name())
		return
	}
	fields := make([]values.Value, structVal.StructType().NumFields())
	for i, field := range structVal.StructType().Fields.Fields() {
		fields[i] = structVal.FieldValue(field.Name.Name)
	}

	field0Value, ok := fields[0].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", fields[0], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	field0 := types.NewAtom[int32](field0Value)

	s.Val = field0
	return
}

func (s InSlice) String() string {
	return s.handle.String()
}

// Bridge returns the bridge between the Go value and the GX value.
func (s *InSlice) Bridge() types.Bridge { return &s.handle }

// handleStruct stores the backend handles of Struct.
type handleStruct struct {
	pkg   *Package
	struc *ir.NamedType
	owner *Struct

	runnerSetNotInSlice *MethodStructSetNotInSlice
}

// MethodStructSetNotInSlice compiles and runs the GX function SetNotInSlice for a device.
// SetNotInSlice tests setting a structure field with a structure.
type MethodStructSetNotInSlice struct {
	methodBase
	receiver handleStruct
}

// Run first compiles SetNotInSlice for a given device and the given arguments.
// Once compiled, the function is then run with these same arguments.
// If the shape of the arguments change, the function will panic.
func (f *MethodStructSetNotInSlice) Run(arg0 *NotInSlice) (_ *Struct, err error) {
	var args []values.Value = []values.Value{
		arg0.Bridge().GXValue(), // d NotInSlice
	}
	if f.runner == nil {
		f.runner, err = tracer.Trace(f.pkg.Device, f.function.(*ir.FuncDecl), f.receiver.GXValue(), args, f.pkg.options)
		if err != nil {
			return
		}
	}
	var outputs []values.Value
	outputs, err = f.runner.Run(f.receiver.GXValue(), args, f.pkg.Package.Tracer)
	if err != nil {
		return
	}

	cmpl := f.pkg
	var out0 *Struct
	out0, err = cmpl.MarshalStruct(outputs[0])
	if err != nil {
		return
	}

	return out0, nil
}

func (f *MethodStructSetNotInSlice) String() string {
	return fmt.Sprint(f.function)
}

// Type of the value.
func (h *handleStruct) Type() ir.Type {
	return h.struc
}

// NamedType returns the intermediate representation of the type.
func (h *handleStruct) NamedType() *ir.NamedType {
	return h.struc
}

// Bridger returns the Go object owning this handle.
func (h *handleStruct) Bridger() types.Bridger {
	return h.owner
}

// GXValue returns the GX value.
func (h *handleStruct) GXValue() values.Value {
	return h.owner.value
}

// String representation of the handle.
func (h *handleStruct) String() string {
	bld := strings.Builder{}
	bld.WriteString("Struct{\n")

	bld.WriteString(fmt.Sprintf("%s:%s\n", "A", any(h.owner.A).(fmt.Stringer).String()))

	bld.WriteString(fmt.Sprintf("%s:%s\n", "B", any(h.owner.B).(fmt.Stringer).String()))

	bld.WriteString(fmt.Sprintf("%s:%s\n", "C", any(h.owner.C).(fmt.Stringer).String()))

	bld.WriteString(fmt.Sprintf("%s:%s\n", "D", any(h.owner.D).(fmt.Stringer).String()))

	bld.WriteString(fmt.Sprintf("%s:%s\n", "specialIndex", any(h.owner.specialIndex).(fmt.Stringer).String()))

	bld.WriteString(fmt.Sprintf("%s:%s\n", "SpecialValue", any(h.owner.SpecialValue).(fmt.Stringer).String()))

	bld.WriteString("}")
	return bld.String()
}

// Struct stores the handle of Struct on a device.
type Struct struct {
	handle handleStruct
	value  *values.NamedType

	A types.Array[float32]

	B *types.Slice[types.Atom[float32]]

	C *types.Slice[*InSlice]

	D *NotInSlice

	specialIndex types.Atom[int32]

	SpecialValue types.Atom[float32]
}

var (
	_ types.Bridger      = (*Struct)(nil)
	_ types.StructBridge = (*handleStruct)(nil)
)

// StructValue returns the GX value of the structure.
func (h *handleStruct) StructValue() *values.Struct {
	return h.owner.value.Underlying().(*values.Struct)
}

// MarshalStruct populates the receiver fields with device handles.
func (cmpl *Package) MarshalStruct(val values.Value) (s *Struct, err error) {
	s = cmpl.Factory.NewStruct()
	var ok bool
	s.value, ok = val.(*values.NamedType)
	if !ok {
		err = errors.Errorf("cannot use handle to set Struct: %T is not a %s", val, reflect.TypeFor[*values.NamedType]())
		return
	}
	structVal, ok := s.value.Underlying().(*values.Struct)
	if !ok {
		err = errors.Errorf("incorrect underlying value for named type Struct: %T is not a %s", val, reflect.TypeFor[*values.Struct]().Name())
		return
	}
	fields := make([]values.Value, structVal.StructType().NumFields())
	for i, field := range structVal.StructType().Fields.Fields() {
		fields[i] = structVal.FieldValue(field.Name.Name)
	}

	field0Value, ok := fields[0].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", fields[0], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	field0 := types.NewArray[float32](field0Value)

	field1Slice, ok := fields[1].(*values.Slice)
	if !ok {
		err = fmt.Errorf("cannot use value %T to set []<no value>: not a slice", fields[1])
		return
	}
	field1Elements := make([]types.Atom[float32], field1Slice.Len())
	for i := 0; i < field1Slice.Len(); i++ {
		field1HandleI := field1Slice.Element(i)

		field1ElmtIValue, ok := field1HandleI.(values.Array)
		if !ok {
			err = errors.Errorf("cannot cast %T to %s", field1HandleI, reflect.TypeFor[*values.DeviceArray]().Name())
			return
		}
		field1ElmtI := types.NewAtom[float32](field1ElmtIValue)

		field1Elements[i] = field1ElmtI
	}
	field1, err := types.NewSlice[types.Atom[float32]](
		field1Slice.SliceType(),
		field1Elements,
	)
	if err != nil {
		return nil, err
	}

	field2Slice, ok := fields[2].(*values.Slice)
	if !ok {
		err = fmt.Errorf("cannot use value %T to set []<no value>: not a slice", fields[2])
		return
	}
	field2Elements := make([]*InSlice, field2Slice.Len())
	for i := 0; i < field2Slice.Len(); i++ {
		field2HandleI := field2Slice.Element(i)
		var field2ElmtI *InSlice
		field2ElmtI, err = cmpl.MarshalInSlice(field2HandleI)
		if err != nil {
			return
		}
		field2Elements[i] = field2ElmtI
	}
	field2, err := types.NewSlice[*InSlice](
		field2Slice.SliceType(),
		field2Elements,
	)
	if err != nil {
		return nil, err
	}

	var field3 *NotInSlice
	field3, err = cmpl.MarshalNotInSlice(fields[3])
	if err != nil {
		return
	}

	field4Value, ok := fields[4].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", fields[4], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	field4 := types.NewAtom[int32](field4Value)

	field5Value, ok := fields[5].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", fields[5], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	field5 := types.NewAtom[float32](field5Value)

	s.A = field0
	s.B = field1
	s.C = field2
	s.D = field3
	s.specialIndex = field4
	s.SpecialValue = field5
	return
}

func (s Struct) String() string {
	return s.handle.String()
}

// Bridge returns the bridge between the Go value and the GX value.
func (s *Struct) Bridge() types.Bridge { return &s.handle }

// SetNotInSlice returns a handle to compile method SetNotInSlice for a device.
func (s Struct) SetNotInSlice() *MethodStructSetNotInSlice {
	return s.handle.runnerSetNotInSlice
}

type methodBase struct {
	pkg      *Package
	function ir.Func
	runner   tracer.CompiledFunc
}

// NewStruct compiles and runs the GX function NewStruct for a device.
// New returns a new structure.
type NewStruct struct {
	methodBase
}

// AddToStruct compiles and runs the GX function AddToStruct for a device.
// AddToStruct adds a scalar to the structure field.
type AddToStruct struct {
	methodBase
}

// AddFloat32 compiles and runs the GX function AddFloat32 for a device.
// AddFloat32 adds x and y.
type AddFloat32 struct {
	methodBase
}

// AddInt compiles and runs the GX function AddInt for a device.
// AddInt adds x and y.
type AddInt struct {
	methodBase
}

// AddFloat32s compiles and runs the GX function AddFloat32s for a device.
// Add x and y.
type AddFloat32s struct {
	methodBase
}

// AddInts compiles and runs the GX function AddInts for a device.
// AddInts x and y.
type AddInts struct {
	methodBase
}

// Len compiles and runs the GX function Len for a device.
// Len returns the outmost dimension of x.
type Len struct {
	methodBase
}

// Iota compiles and runs the GX function Iota for a device.
// Iota returns an array filled with numbers.
type Iota struct {
	methodBase
}

// SliceArrayArgConstIndex compiles and runs the GX function SliceArrayArgConstIndex for a device.
// SliceArrayArg checks that we can slice an array type argument.
type SliceArrayArgConstIndex struct {
	methodBase
}

// SliceArrayArg compiles and runs the GX function SliceArrayArg for a device.
// SliceArrayArg checks that we can slice an array type argument.
type SliceArrayArg struct {
	methodBase
}

// SliceSliceArg compiles and runs the GX function SliceSliceArg for a device.
// SliceSliceArg checks that we can slice a slice.
type SliceSliceArg struct {
	methodBase
}

// NewNotInSlice compiles and runs the GX function NewNotInSlice for a device.
// NewNotInSlice returns a NotInSlice instance with its Val attributes assigned to val.
type NewNotInSlice struct {
	methodBase
}

// Run first compiles NewStruct for a given device and the given arguments.
// Once compiled, the function is then run with these same arguments.
// If the shape of the arguments change, the function will panic.
func (f *NewStruct) Run(arg0 types.Atom[float32]) (_ *Struct, err error) {
	var args []values.Value = []values.Value{
		arg0.Bridge().GXValue(), // offset float32
	}
	if f.runner == nil {
		f.runner, err = tracer.Trace(f.pkg.Device, f.function.(*ir.FuncDecl), nil, args, f.pkg.options)
		if err != nil {
			return
		}
	}
	var outputs []values.Value
	outputs, err = f.runner.Run(nil, args, f.pkg.Package.Tracer)
	if err != nil {
		return
	}

	cmpl := f.pkg
	var out0 *Struct
	out0, err = cmpl.MarshalStruct(outputs[0])
	if err != nil {
		return
	}

	return out0, nil
}

func (f *NewStruct) String() string {
	return fmt.Sprint(f.function)
}

// Run first compiles AddToStruct for a given device and the given arguments.
// Once compiled, the function is then run with these same arguments.
// If the shape of the arguments change, the function will panic.
func (f *AddToStruct) Run(arg0 *Struct) (_ *Struct, err error) {
	var args []values.Value = []values.Value{
		arg0.Bridge().GXValue(), // a Struct
	}
	if f.runner == nil {
		f.runner, err = tracer.Trace(f.pkg.Device, f.function.(*ir.FuncDecl), nil, args, f.pkg.options)
		if err != nil {
			return
		}
	}
	var outputs []values.Value
	outputs, err = f.runner.Run(nil, args, f.pkg.Package.Tracer)
	if err != nil {
		return
	}

	cmpl := f.pkg
	var out0 *Struct
	out0, err = cmpl.MarshalStruct(outputs[0])
	if err != nil {
		return
	}

	return out0, nil
}

func (f *AddToStruct) String() string {
	return fmt.Sprint(f.function)
}

// Run first compiles AddFloat32 for a given device and the given arguments.
// Once compiled, the function is then run with these same arguments.
// If the shape of the arguments change, the function will panic.
func (f *AddFloat32) Run(arg0 types.Atom[float32], arg1 types.Atom[float32]) (_ types.Atom[float32], err error) {
	var args []values.Value = []values.Value{
		arg0.Bridge().GXValue(), // x float32
		arg1.Bridge().GXValue(), // y float32
	}
	if f.runner == nil {
		f.runner, err = tracer.Trace(f.pkg.Device, f.function.(*ir.FuncDecl), nil, args, f.pkg.options)
		if err != nil {
			return
		}
	}
	var outputs []values.Value
	outputs, err = f.runner.Run(nil, args, f.pkg.Package.Tracer)
	if err != nil {
		return
	}

	out0Value, ok := outputs[0].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", outputs[0], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	out0 := types.NewAtom[float32](out0Value)

	return out0, nil
}

func (f *AddFloat32) String() string {
	return fmt.Sprint(f.function)
}

// Run first compiles AddInt for a given device and the given arguments.
// Once compiled, the function is then run with these same arguments.
// If the shape of the arguments change, the function will panic.
func (f *AddInt) Run(arg0 types.Atom[int64], arg1 types.Atom[int64]) (_ types.Atom[int64], err error) {
	var args []values.Value = []values.Value{
		arg0.Bridge().GXValue(), // x int64
		arg1.Bridge().GXValue(), // y int64
	}
	if f.runner == nil {
		f.runner, err = tracer.Trace(f.pkg.Device, f.function.(*ir.FuncDecl), nil, args, f.pkg.options)
		if err != nil {
			return
		}
	}
	var outputs []values.Value
	outputs, err = f.runner.Run(nil, args, f.pkg.Package.Tracer)
	if err != nil {
		return
	}

	out0Value, ok := outputs[0].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", outputs[0], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	out0 := types.NewAtom[int64](out0Value)

	return out0, nil
}

func (f *AddInt) String() string {
	return fmt.Sprint(f.function)
}

// Run first compiles AddFloat32s for a given device and the given arguments.
// Once compiled, the function is then run with these same arguments.
// If the shape of the arguments change, the function will panic.
func (f *AddFloat32s) Run(arg0 types.Array[float32], arg1 types.Array[float32]) (_ types.Array[float32], err error) {
	var args []values.Value = []values.Value{
		arg0.Bridge().GXValue(), // x [a]float32
		arg1.Bridge().GXValue(), // y [a]float32
	}
	if f.runner == nil {
		f.runner, err = tracer.Trace(f.pkg.Device, f.function.(*ir.FuncDecl), nil, args, f.pkg.options)
		if err != nil {
			return
		}
	}
	var outputs []values.Value
	outputs, err = f.runner.Run(nil, args, f.pkg.Package.Tracer)
	if err != nil {
		return
	}

	out0Value, ok := outputs[0].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", outputs[0], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	out0 := types.NewArray[float32](out0Value)

	return out0, nil
}

func (f *AddFloat32s) String() string {
	return fmt.Sprint(f.function)
}

// Run first compiles AddInts for a given device and the given arguments.
// Once compiled, the function is then run with these same arguments.
// If the shape of the arguments change, the function will panic.
func (f *AddInts) Run(arg0 types.Array[int64], arg1 types.Array[int64]) (_ types.Array[int64], err error) {
	var args []values.Value = []values.Value{
		arg0.Bridge().GXValue(), // x [a]int64
		arg1.Bridge().GXValue(), // y [a]int64
	}
	if f.runner == nil {
		f.runner, err = tracer.Trace(f.pkg.Device, f.function.(*ir.FuncDecl), nil, args, f.pkg.options)
		if err != nil {
			return
		}
	}
	var outputs []values.Value
	outputs, err = f.runner.Run(nil, args, f.pkg.Package.Tracer)
	if err != nil {
		return
	}

	out0Value, ok := outputs[0].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", outputs[0], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	out0 := types.NewArray[int64](out0Value)

	return out0, nil
}

func (f *AddInts) String() string {
	return fmt.Sprint(f.function)
}

// Run first compiles Len for a given device and the given arguments.
// Once compiled, the function is then run with these same arguments.
// If the shape of the arguments change, the function will panic.
func (f *Len) Run(arg0 types.Array[float32]) (_ types.Atom[int64], err error) {
	var args []values.Value = []values.Value{
		arg0.Bridge().GXValue(), // x []float32
	}
	if f.runner == nil {
		f.runner, err = tracer.Trace(f.pkg.Device, f.function.(*ir.FuncDecl), nil, args, f.pkg.options)
		if err != nil {
			return
		}
	}
	var outputs []values.Value
	outputs, err = f.runner.Run(nil, args, f.pkg.Package.Tracer)
	if err != nil {
		return
	}

	out0Value, ok := outputs[0].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", outputs[0], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	out0 := types.NewAtom[int64](out0Value)

	return out0, nil
}

func (f *Len) String() string {
	return fmt.Sprint(f.function)
}

// Run first compiles Iota for a given device and the given arguments.
// Once compiled, the function is then run with these same arguments.
// If the shape of the arguments change, the function will panic.
func (f *Iota) Run() (_ types.Array[int64], err error) {
	var args []values.Value = nil
	if f.runner == nil {
		f.runner, err = tracer.Trace(f.pkg.Device, f.function.(*ir.FuncDecl), nil, args, f.pkg.options)
		if err != nil {
			return
		}
	}
	var outputs []values.Value
	outputs, err = f.runner.Run(nil, args, f.pkg.Package.Tracer)
	if err != nil {
		return
	}

	out0Value, ok := outputs[0].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", outputs[0], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	out0 := types.NewArray[int64](out0Value)

	return out0, nil
}

func (f *Iota) String() string {
	return fmt.Sprint(f.function)
}

// Run first compiles SliceArrayArgConstIndex for a given device and the given arguments.
// Once compiled, the function is then run with these same arguments.
// If the shape of the arguments change, the function will panic.
func (f *SliceArrayArgConstIndex) Run(arg0 types.Array[float32]) (_ types.Array[float32], _ types.Array[float32], _ types.Array[float32], err error) {
	var args []values.Value = []values.Value{
		arg0.Bridge().GXValue(), // a [3][2]float32
	}
	if f.runner == nil {
		f.runner, err = tracer.Trace(f.pkg.Device, f.function.(*ir.FuncDecl), nil, args, f.pkg.options)
		if err != nil {
			return
		}
	}
	var outputs []values.Value
	outputs, err = f.runner.Run(nil, args, f.pkg.Package.Tracer)
	if err != nil {
		return
	}

	out0Value, ok := outputs[0].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", outputs[0], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	out0 := types.NewArray[float32](out0Value)

	out1Value, ok := outputs[1].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", outputs[1], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	out1 := types.NewArray[float32](out1Value)

	out2Value, ok := outputs[2].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", outputs[2], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	out2 := types.NewArray[float32](out2Value)

	return out0, out1, out2, nil
}

func (f *SliceArrayArgConstIndex) String() string {
	return fmt.Sprint(f.function)
}

// Run first compiles SliceArrayArg for a given device and the given arguments.
// Once compiled, the function is then run with these same arguments.
// If the shape of the arguments change, the function will panic.
func (f *SliceArrayArg) Run(arg0 types.Array[float32], arg1 types.Atom[int32]) (_ types.Array[float32], err error) {
	var args []values.Value = []values.Value{
		arg0.Bridge().GXValue(), // a [3][2]float32
		arg1.Bridge().GXValue(), // i int32
	}
	if f.runner == nil {
		f.runner, err = tracer.Trace(f.pkg.Device, f.function.(*ir.FuncDecl), nil, args, f.pkg.options)
		if err != nil {
			return
		}
	}
	var outputs []values.Value
	outputs, err = f.runner.Run(nil, args, f.pkg.Package.Tracer)
	if err != nil {
		return
	}

	out0Value, ok := outputs[0].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", outputs[0], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	out0 := types.NewArray[float32](out0Value)

	return out0, nil
}

func (f *SliceArrayArg) String() string {
	return fmt.Sprint(f.function)
}

// Run first compiles SliceSliceArg for a given device and the given arguments.
// Once compiled, the function is then run with these same arguments.
// If the shape of the arguments change, the function will panic.
func (f *SliceSliceArg) Run(arg0 *types.Slice[types.Array[float32]], arg1 types.Atom[int32]) (_ types.Array[float32], err error) {
	var args []values.Value = []values.Value{
		arg0.Bridge().GXValue(), // a [][2]float32
		arg1.Bridge().GXValue(), // i int32
	}
	if f.runner == nil {
		f.runner, err = tracer.Trace(f.pkg.Device, f.function.(*ir.FuncDecl), nil, args, f.pkg.options)
		if err != nil {
			return
		}
	}
	var outputs []values.Value
	outputs, err = f.runner.Run(nil, args, f.pkg.Package.Tracer)
	if err != nil {
		return
	}

	out0Value, ok := outputs[0].(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", outputs[0], reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	out0 := types.NewArray[float32](out0Value)

	return out0, nil
}

func (f *SliceSliceArg) String() string {
	return fmt.Sprint(f.function)
}

// Run first compiles NewNotInSlice for a given device and the given arguments.
// Once compiled, the function is then run with these same arguments.
// If the shape of the arguments change, the function will panic.
func (f *NewNotInSlice) Run(arg0 types.Atom[int32]) (_ *NotInSlice, err error) {
	var args []values.Value = []values.Value{
		arg0.Bridge().GXValue(), // val int32
	}
	if f.runner == nil {
		f.runner, err = tracer.Trace(f.pkg.Device, f.function.(*ir.FuncDecl), nil, args, f.pkg.options)
		if err != nil {
			return
		}
	}
	var outputs []values.Value
	outputs, err = f.runner.Run(nil, args, f.pkg.Package.Tracer)
	if err != nil {
		return
	}

	cmpl := f.pkg
	var out0 *NotInSlice
	out0, err = cmpl.MarshalNotInSlice(outputs[0])
	if err != nil {
		return
	}

	return out0, nil
}

func (f *NewNotInSlice) String() string {
	return fmt.Sprint(f.function)
}

// NewNotInSlice returns a handle on named type NotInSlice.
func (fac *Factory) NewNotInSlice() *NotInSlice {
	s := &NotInSlice{}
	typ := fac.Package.Package.IR.Decls.TypeByName("NotInSlice")
	s.handle = handleNotInSlice{
		pkg:   fac.Package,
		struc: typ,
		owner: s,
	}

	structVal, err := values.NewStruct(typ, nil)
	if err != nil {
		panic(err)
	}
	s.value = values.NewNamedType(structVal, typ)

	return s
}

var _ types.Bridge = (*handleNotInSlice)(nil)

func (h *handleNotInSlice) NewFromField(field *ir.Field) (types.Bridge, error) {
	name := field.Name.Name
	switch name {

	case "Val":
		return nil, errors.Errorf("cannot create a new instance for field Val: type types.Atom[int32] not supported")

	default:
		return nil, errors.Errorf("structure NotInSlice has no field %q", name)
	}
}

// SetField sets a field in the structure.
func (h *handleNotInSlice) SetField(field *ir.Field, val types.Bridge) error {

	name := field.Name.Name
	structVal, ok := h.owner.value.Underlying().(*values.Struct)
	if !ok {
		return fmt.Errorf("incorrect underlying value for named type NotInSlice: %T is not a %s", val, reflect.TypeFor[*values.Struct]().Name())
	}
	switch name {

	case "Val":
		bridger := val.Bridger()
		fieldValue, ok := bridger.(types.Atom[int32])
		if !ok {
			return errors.Errorf("cannot set field Val: cannot cast %T to types.Atom[int32]", bridger)
		}
		h.owner.Val = fieldValue
		structVal.SetField("Val", val.GXValue())
		return nil

	default:
		return errors.Errorf("structure NotInSlice has no field %q", name)
	}

}

// NewInSlice returns a handle on named type InSlice.
func (fac *Factory) NewInSlice() *InSlice {
	s := &InSlice{}
	typ := fac.Package.Package.IR.Decls.TypeByName("InSlice")
	s.handle = handleInSlice{
		pkg:   fac.Package,
		struc: typ,
		owner: s,
	}

	structVal, err := values.NewStruct(typ, nil)
	if err != nil {
		panic(err)
	}
	s.value = values.NewNamedType(structVal, typ)

	return s
}

var _ types.Bridge = (*handleInSlice)(nil)

func (h *handleInSlice) NewFromField(field *ir.Field) (types.Bridge, error) {
	name := field.Name.Name
	switch name {

	case "Val":
		return nil, errors.Errorf("cannot create a new instance for field Val: type types.Atom[int32] not supported")

	default:
		return nil, errors.Errorf("structure InSlice has no field %q", name)
	}
}

// SetField sets a field in the structure.
func (h *handleInSlice) SetField(field *ir.Field, val types.Bridge) error {

	name := field.Name.Name
	structVal, ok := h.owner.value.Underlying().(*values.Struct)
	if !ok {
		return fmt.Errorf("incorrect underlying value for named type InSlice: %T is not a %s", val, reflect.TypeFor[*values.Struct]().Name())
	}
	switch name {

	case "Val":
		bridger := val.Bridger()
		fieldValue, ok := bridger.(types.Atom[int32])
		if !ok {
			return errors.Errorf("cannot set field Val: cannot cast %T to types.Atom[int32]", bridger)
		}
		h.owner.Val = fieldValue
		structVal.SetField("Val", val.GXValue())
		return nil

	default:
		return errors.Errorf("structure InSlice has no field %q", name)
	}

}

// NewStruct returns a handle on named type Struct.
func (fac *Factory) NewStruct() *Struct {
	s := &Struct{}
	typ := fac.Package.Package.IR.Decls.TypeByName("Struct")
	s.handle = handleStruct{
		pkg:   fac.Package,
		struc: typ,
		owner: s,
	}

	structVal, err := values.NewStruct(typ, nil)
	if err != nil {
		panic(err)
	}
	s.value = values.NewNamedType(structVal, typ)

	s.handle.runnerSetNotInSlice = &MethodStructSetNotInSlice{
		methodBase: s.handle.pkg.methodStructSetNotInSlice,
		receiver:   s.handle,
	}

	return s
}

var _ types.Bridge = (*handleStruct)(nil)

func (h *handleStruct) NewFromField(field *ir.Field) (types.Bridge, error) {
	name := field.Name.Name
	switch name {

	case "A":
		return nil, errors.Errorf("cannot create a new instance for field A: type types.Array[float32] not supported")

	case "B":
		slice, err := types.NewEmptySlice[types.Atom[float32]](field.Type(), nil)
		if err != nil {
			return nil, err
		}
		return slice.Bridge(), nil

	case "C":
		slice, err := types.NewEmptySlice[*InSlice](field.Type(), func() (types.Bridge, error) {
			return h.pkg.Factory.NewInSlice().Bridge(), nil
		})
		if err != nil {
			return nil, err
		}
		return slice.Bridge(), nil

	case "D":
		return h.pkg.Factory.NewNotInSlice().Bridge(), nil

	case "specialIndex":
		return nil, errors.Errorf("cannot create a new instance for field specialIndex: type types.Atom[int32] not supported")

	case "SpecialValue":
		return nil, errors.Errorf("cannot create a new instance for field SpecialValue: type types.Atom[float32] not supported")

	default:
		return nil, errors.Errorf("structure Struct has no field %q", name)
	}
}

// SetField sets a field in the structure.
func (h *handleStruct) SetField(field *ir.Field, val types.Bridge) error {

	name := field.Name.Name
	structVal, ok := h.owner.value.Underlying().(*values.Struct)
	if !ok {
		return fmt.Errorf("incorrect underlying value for named type Struct: %T is not a %s", val, reflect.TypeFor[*values.Struct]().Name())
	}
	switch name {

	case "A":
		bridger := val.Bridger()
		fieldValue, ok := bridger.(types.Array[float32])
		if !ok {
			return errors.Errorf("cannot set field A: cannot cast %T to types.Array[float32]", bridger)
		}
		h.owner.A = fieldValue
		structVal.SetField("A", val.GXValue())
		return nil

	case "B":
		bridger := val.Bridger()
		fieldValue, ok := bridger.(*types.Slice[types.Atom[float32]])
		if !ok {
			return errors.Errorf("cannot set field B: cannot cast %T to *types.Slice[types.Atom[float32]]", bridger)
		}
		h.owner.B = fieldValue
		structVal.SetField("B", val.GXValue())
		return nil

	case "C":
		bridger := val.Bridger()
		fieldValue, ok := bridger.(*types.Slice[*InSlice])
		if !ok {
			return errors.Errorf("cannot set field C: cannot cast %T to *types.Slice[*InSlice]", bridger)
		}
		h.owner.C = fieldValue
		structVal.SetField("C", val.GXValue())
		return nil

	case "D":
		bridger := val.Bridger()
		fieldValue, ok := bridger.(*NotInSlice)
		if !ok {
			return errors.Errorf("cannot set field D: cannot cast %T to *NotInSlice", bridger)
		}
		h.owner.D = fieldValue
		structVal.SetField("D", val.GXValue())
		return nil

	case "specialIndex":
		bridger := val.Bridger()
		fieldValue, ok := bridger.(types.Atom[int32])
		if !ok {
			return errors.Errorf("cannot set field specialIndex: cannot cast %T to types.Atom[int32]", bridger)
		}
		h.owner.specialIndex = fieldValue
		structVal.SetField("specialIndex", val.GXValue())
		return nil

	case "SpecialValue":
		bridger := val.Bridger()
		fieldValue, ok := bridger.(types.Atom[float32])
		if !ok {
			return errors.Errorf("cannot set field SpecialValue: cannot cast %T to types.Atom[float32]", bridger)
		}
		h.owner.SpecialValue = fieldValue
		structVal.SetField("SpecialValue", val.GXValue())
		return nil

	default:
		return errors.Errorf("structure Struct has no field %q", name)
	}

}
