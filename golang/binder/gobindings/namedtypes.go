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

package gobindings

import (
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/base/tmpl"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/binder/bindings"
)

type namedType interface {
	named() *ir.NamedType
	BuildDeclaration() (string, error)
	Methods() []method
	Index() int
	InitGXValue() (string, error)
	Fields() []structField
}

func buildTypes(b *binder) ([]namedType, error) {
	var types []namedType
	for i, gxType := range b.Package.Decls.Types {
		_, ok := gxType.Underlying.Typ.(*ir.BuiltinType)
		if ok {
			continue
		}
		typ, err := b.buildType(i, gxType)
		if err != nil {
			return nil, err
		}
		types = append(types, typ)
	}
	return types, nil
}

func (b *binder) buildType(index int, gxType *ir.NamedType) (namedType, error) {
	switch typT := gxType.Underlying.Typ.(type) {
	case ir.ArrayType:
		if !typT.Rank().IsAtomic() {
			return nil, errors.Errorf("array named type %s:%T not supported", gxType.Name(), typT)
		}
		atom := &scalarType{
			baseType: baseType{
				binder: b,
				Named:  gxType,
				index:  index,
			},
			typ: typT,
		}
		atom.methods = b.buildMethods(atom)
		return atom, nil
	case *ir.StructType:
		struc := &structType{
			baseType: baseType{
				binder: b,
				Named:  gxType,
				index:  index,
			},
			typ: typT,
		}
		var err error
		struc.fields, err = struc.buildFields()
		if err != nil {
			return nil, err
		}
		struc.methods = b.buildMethods(struc)
		return struc, nil
	default:
		return nil, errors.Errorf("cannot write bindings for named type %s:%T", gxType.Name(), typT)
	}
}

const namedTypeHandle = `
// handle{{.Named.Name}} stores the backend handles of {{.Named.Name}}.
type handle{{.Named.Name}} struct {
	pkg       *Package
	struc     *ir.NamedType
	owner     *{{.Named.Name}}
}

// Type of the value.
func (h *handle{{.Named.Name}}) Type() ir.Type {
	return h.struc
}

// NamedType returns the intermediate representation of the type.
func (h *handle{{.Named.Name}}) NamedType() *ir.NamedType {
	return h.struc
}

// Bridger returns the Go object owning this handle.
func (h *handle{{.Named.Name}}) Bridger() types.Bridger {
	return h.owner
}

// GXValue returns the GX value.
func (h *handle{{.Named.Name}}) GXValue() values.Value {
	return h.owner.value
}

// String representation of the handle.
func (h *handle{{.Named.Name}}) String() string {
	bld := strings.Builder{}
	bld.WriteString("{{.Named.Name}}{\n")
{{.NamedTypeFieldsToString}}
	bld.WriteString("}")
	return bld.String()
}
`

type baseType struct {
	*binder
	Named *ir.NamedType
	index int

	CannotBeOnDevice string
	methods          []method
}

func (s baseType) named() *ir.NamedType {
	return s.Named
}

func (s baseType) Methods() []method {
	return s.methods
}

func (s baseType) Index() int {
	return s.index
}

type scalarType struct {
	baseType
	typ ir.ArrayType
}

var (
	scalarBackendTemplate = template.Must(template.New("namedScalarTypeTMPL").Parse(namedTypeHandle + `
// {{.Named.Name}} stores the handle of {{.Named.Name}} on a backend.
type {{.Named.Name}} struct {
	value {{.GXValueType}}
	handle handle{{.Named.Name}}
}

var (
	_ types.Bridger = (*{{.Named.Name}})(nil)
	_ types.Bridge = (*handle{{.Named.Name}})(nil)
)

func (val {{.Named.Name}}) String() string {
	return val.handle.String()
}

// Bridge returns the bridge between the Go value and the GX value.
func (val *{{.Named.Name}}) Bridge() types.Bridge { return &val.handle }

// Marshal{{.Named.Name}} populates the receiver fields with device handles.
func (fty *Factory) Marshal{{.Named.Name}}(val values.Value) (s *{{.Named.Name}}, err error) {
	s = fty.New{{.Named.Name}}()
	if _, ok := val.(*values.Slice); ok {
		err = fmt.Errorf("cannot use handle to set {{.Named.Name}}: got a tuple instead of a single value")
		return
	}
	s.value = val.({{.GXValueType}})
	return
}

`))

	scalarNotSupportedTemplate = template.Must(template.New("scalarNotDeviceTMPL").Parse(`
// {{.Named.Name}} type not generated:
// {{.CannotBeOnDevice}}.
`))
)

func (s scalarType) BuildDeclaration() (string, error) {
	tmpl := scalarBackendTemplate
	if err := bindings.CanBeOnDevice(s.typ); err != nil {
		s.CannotBeOnDevice = err.Error()
		tmpl = scalarNotSupportedTemplate
	}
	var w strings.Builder
	if err := tmpl.Execute(&w, s); err != nil {
		return "", err
	}
	return w.String(), nil
}

func (s scalarType) BackendType() (string, error) {
	return s.binder.bridgerType(s.typ)
}

func (s scalarType) GXValueType() (string, error) {
	return s.binder.gxValueType(s.typ)
}

func (s scalarType) NamedTypeFieldsToString() (string, error) {
	return "\tbld.WriteString(h.owner.value.(fmt.Stringer).String())", nil
}

func (s scalarType) Fields() []structField {
	return nil
}

func (s scalarType) InitGXValue() (string, error) {
	return "", nil
}

type (
	structType struct {
		baseType
		typ    *ir.StructType
		fields []structField
	}

	structField struct {
		structType  *structType
		constructor *template.Template

		Name        string
		BridgerType string
		Field       *ir.Field

		SliceElementFactory     string
		SliceElementBridgerType string

		Factory string
	}
)

var (
	fieldNotSupported = template.Must(template.New("fieldNotSupportedTMPL").Parse(strings.TrimSpace(`
		return nil, errors.Errorf("cannot create a new instance for field {{.Field.Name}}: type {{.BridgerType}} not supported")
	`)))

	fieldNamedType = template.Must(template.New("fieldNamedTypeTMPL").Parse(strings.TrimSpace(`
		return h.pkg.{{.Factory}}.New{{.Field.Type.Name}}().Bridge(), nil
	`)))

	fieldSliceType = template.Must(template.New("fieldSliceTypeTMPL").Parse(strings.TrimSpace(`
		slice, err := types.NewEmptySlice[{{.SliceElementBridgerType}}](field.Type(), {{.SliceElementFactory}})
		if err != nil {
			return nil, err
		}
		return slice.Bridge(), nil
	`)))
)

func (s *structType) sliceElementFactory(typ *ir.SliceType) (string, error) {
	namedType, ok := typ.DType.Typ.(*ir.NamedType)
	if !ok {
		return "nil", nil
	}
	factory, err := s.namedTypeFactory(namedType)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`func () (types.Bridge, error) {
		return h.pkg.%s.New%s().Bridge(), nil
	}`, factory, namedType.Name()), nil
}

func (s *structType) buildFields() ([]structField, error) {
	fields := s.typ.Fields.Fields()
	var structFields []structField
	for _, field := range fields {
		goType, err := s.binder.bridgerType(field.Group.Type.Typ)
		if err != nil {
			return nil, err
		}
		sField := structField{
			Name:        field.Name.Name,
			BridgerType: goType,
			Field:       field,

			constructor: fieldNotSupported,
		}
		switch typT := field.Type().(type) {
		case *ir.NamedType:
			sField.constructor = fieldNamedType
			sField.Factory, err = s.namedTypeFactory(typT)
			if err != nil {
				return nil, err
			}

		case *ir.SliceType:
			sField.constructor = fieldSliceType
			sField.SliceElementBridgerType, err = s.bridgerType(typT.DType.Typ)
			if err != nil {
				return nil, err
			}
			sField.SliceElementFactory, err = s.sliceElementFactory(typT)
			if err != nil {
				return nil, err
			}
		}
		structFields = append(structFields, sField)
	}
	return structFields, nil
}

func (s structField) Constructor() (string, error) {
	return tmplExecute(s.constructor, s)
}

var (
	structTemplate = template.Must(template.New("structTMPL").Parse(namedTypeHandle + `
// {{.Named.Name}} stores the handle of {{.Named.Name}} on a device.
type {{.Named.Name}} struct {
	handle handle{{.Named.Name}}
	value *values.NamedType

{{range $field := .Fields -}}
	{{$field.Name}} {{$field.BridgerType}}
{{end -}}
}

var (
	_ types.Bridger = (*{{.Named.Name}})(nil)
	_ types.StructBridge = (*handle{{.Named.Name}})(nil)
)

// StructValue returns the GX value of the structure.
func (h *handle{{.Named.Name}}) StructValue() *values.Struct {
	return h.owner.value.Underlying().(*values.Struct)
}

// Marshal{{.Named.Name}} populates the receiver fields with device handles.
func (fty *Factory) Marshal{{.Named.Name}}(val values.Value) (s *{{.Named.Name}}, err error) {
	s = fty.New{{.Named.Name}}()
	var ok bool
	s.value, ok = val.(*values.NamedType)
	if !ok {
		err = errors.Errorf("cannot use handle to set {{.Named.Name}}: %T is not a %s", val, reflect.TypeFor[*values.NamedType]())
		return
	}
	structVal, ok := s.value.Underlying().(*values.Struct)
	if !ok {
		err = errors.Errorf("incorrect underlying value for named type {{.Named.Name}}: %T is not a %s", val, reflect.TypeFor[*values.Struct]().Name())
		return
	}
	fields := make([]values.Value, structVal.StructType().NumFields())
	for i, field := range structVal.StructType().Fields.Fields() {
		fields[i] = structVal.FieldValue(field.Name.Name)
	}
	{{.SetDeviceFieldsFromSliceValue}}
	return
}

func (s {{.Named.Name}}) String() string {
	return s.handle.String()
}

// Bridge returns the bridge between the Go value and the GX value.
func (s *{{.Named.Name}}) Bridge() types.Bridge { return &s.handle }

`))

	structNotDeviceTemplate = template.Must(template.New("structNotDeviceTMPL").Parse(`
// {{.Named.Name}} type not generated:
// {{.CannotBeOnDevice}}.
`))
)

func (s structType) BuildDeclaration() (string, error) {
	tmpl := structTemplate
	if err := bindings.CanBeOnDevice(s.typ); err != nil {
		s.CannotBeOnDevice = err.Error()
		tmpl = structNotDeviceTemplate
	}
	var w strings.Builder
	if err := tmpl.Execute(&w, s); err != nil {
		return "", err
	}
	return w.String(), nil
}

func (s structType) SetDeviceFieldsFromSliceValue() (string, error) {
	if s.typ.NumFields() == 0 {
		return "", nil
	}
	var res []string
	// Construct all the field values.
	fields := s.typ.Fields.Fields()
	for i, field := range fields {
		fieldSrc := fmt.Sprintf("fields[%d]", i)
		fieldTarget := fmt.Sprintf("field%d", i)
		fieldLines, err := s.setTargetFromSourceType(fieldTarget, fieldSrc, field.Type())
		if err != nil {
			return "", err
		}
		res = slices.Concat(res, fieldLines)
	}
	// Assign field values to their respective field within the structure.
	for i, field := range fields {
		res = append(res, fmt.Sprintf("\ts.%s = field%d", field.Name.Name, i))
	}
	return strings.Join(res, "\n"), nil
}

var fieldToStringTemplate = template.Must(template.New("fieldToStringTMPL").Parse(`
	fmt.Fprintf(&bld, "%s:%s\n", "{{.FieldName}}", any(h.owner.{{.FieldName}}).(fmt.Stringer).String())
`))

func (s structType) NamedTypeFieldsToString() (string, error) {
	fields := s.typ.Fields.Fields()
	return tmpl.IterateFunc(fields, func(i int, field *ir.Field) (string, error) {
		bld := strings.Builder{}
		if err := fieldToStringTemplate.Execute(&bld, map[string]string{
			"FieldName": field.Name.Name,
		}); err != nil {
			return "", err
		}
		return bld.String(), nil
	})
}

var setFieldTemplate = template.Must(template.New("allocStructFieldTMPL").Parse(`
	case "{{.FieldName}}":
		fieldValue, err := h.pkg.handle.factory.New(field.Type().(*ir.NamedType))
		if err != nil {
			return nil, err
		}
		goFieldValue, ok := fieldValue.Bridger().({{.FieldType}})
		if !ok {
			return nil, errors.Errorf("cannot convert %T to %s", fieldValue, reflect.TypeFor[{{.FieldType}}]())
		}
		h.owner.{{.FieldName}} = goFieldValue
		return fieldValue, nil
`))

func (s structType) AllocFields() (string, error) {
	return tmpl.IterateTmpl(s.typ.Fields.Fields(), setFieldTemplate)
}

func (s structType) Fields() []structField {
	return s.fields
}

var initStructGXValueTemplate = template.Must(template.New("initStructGXValueTMPL").Parse(`
	structVal, err := values.NewStruct(typ, nil)
	if err != nil {
		panic(err)
	}
	s.value = values.NewNamedType(structVal, typ)
`))

func (s structType) InitGXValue() (string, error) {
	return tmplExecute(initStructGXValueTemplate, s)
}
