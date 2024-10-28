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

// Package gobindings generates Go bindings for a GX package.
package gobindings

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib"

	_ "embed"
)

//go:embed bindings.go.tmpl
var goBindings string

var goPackageTemplate = template.Must(template.New("GoBindingsTMPL").Parse(goBindings))

func iterateFunc[T any](objs []T, f func(int, T) (string, error)) (string, error) {
	var ss []string
	for i, obj := range objs {
		s, err := f(i, obj)
		if err != nil {
			return "", err
		}
		ss = append(ss, s)
	}
	return strings.Join(ss, "\n"), nil
}

func iterateTmpl[T any](objs []T, tmpl *template.Template) (string, error) {
	buf := strings.Builder{}
	for _, obj := range objs {
		if err := tmpl.Execute(&buf, obj); err != nil {
			return "", errors.Errorf("cannot generate code for %#v: %v", obj, err)
		}
	}
	return buf.String(), nil
}

type (
	// DependenciesBuildCall generates the source code to get the GX package
	// in GX intermediate representation (IR).
	DependenciesBuildCall interface {
		// SourceImport generates the import when the GX source needs to be imported
		// from a Go module.
		SourceImport(*ir.Package) string

		// DependencyImport generates the import to insert in the package source code.
		DependencyImport(string) string

		// StdlibDependencyImport returns the package path to a dependency on
		// the standard library package.
		StdlibDependencyImport(string) string

		// CallBuild generates the source to define and set the irPackage variable
		// used in the source code to return a Package structure.
		CallBuild(*ir.Package) (string, error)
	}

	// binder generates the Go bindings for a GX package.
	binder struct {
		builder DependenciesBuildCall
		Package *ir.Package

		funcs      []*function
		NamedTypes []namedType
		stdlib     *stdlib.Stdlib

		FuncRunners          string
		NamedTypeDefinitions string
		PkgVars              string

		dependencies map[string]dependency
	}
)

func (b *binder) GXImportBindedLibrary() string {
	return b.builder.SourceImport(b.Package)
}

func (b *binder) BuildIRPackage() (string, error) {
	return b.builder.CallBuild(b.Package)
}

var processNamedTypeOutputTmpl = template.Must(template.New("processNamedTypeOutputTMPL").Parse(
	`var {{.Target}} {{.TypePointer}}
{{.Target}}, err = {{.Compiler}}.Marshal{{.TypeName}}({{.Source}})
if err != nil {
	return
}`))

func (b *binder) processNamedTypeOutput(target, src string, typ *ir.NamedType) (res []string, err error) {
	compiler := "cmpl"
	if typ.Package() != b.Package {
		compiler += "." + b.namePackage(typ.Package())
	}
	deviceType, err := b.gxValueTypePointer(typ)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := processNamedTypeOutputTmpl.Execute(&buf, map[string]string{
		"Source":      src,
		"Target":      target,
		"TypeName":    fmt.Sprintf("%s", typ.NameT),
		"TypePointer": deviceType,
		"Compiler":    compiler,
	}); err != nil {
		return nil, err
	}
	return strings.Split(buf.String(), "\n"), nil
}

var setSliceTmpl = template.Must(template.New("setSliceOutputTMPL").Parse(
	`
	{{.Target}}Slice, ok :=  {{.Source}}.(*values.Slice)
	if !ok {
		err = fmt.Errorf("cannot use value %T to set []{{.SliceDType}}: not a slice", {{.Source}})
		return
	}
	{{.Target}}Elements := make([]{{.SliceDataType}}, {{.Target}}Slice.Size())
	for i := 0; i < {{.Target}}Slice.Size(); i++ {
		{{.ElementSource}} := {{.Target}}Slice.Element(i)
		{{.UnmarshalElement}}
		{{.Target}}Elements[i] = {{.ElementTarget}}
	}
	{{.Target}}, err := types.NewSlice[{{.SliceDataType}}](
		{{.Target}}Slice.SliceType(),
		{{.Target}}Elements,
	)
	if err != nil {
		return nil, err
	}
`))

func isNamedStruct(typ ir.Type) bool {
	_, ok := typ.(*ir.NamedType)
	return ok
}

func (b *binder) processSliceOutput(target, src string, typ *ir.SliceType) ([]string, error) {
	sliceDataType, err := b.bridgerType(typ.DType)
	if err != nil {
		return nil, err
	}
	elementSource := target + "HandleI"
	elementTarget := target + "ElmtI"
	setSliceElement, err := b.setTargetFromSourceType(elementTarget, elementSource, typ.DType)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := setSliceTmpl.Execute(&buf, map[string]string{
		"Source":           src,
		"Target":           target,
		"SliceDataType":    sliceDataType,
		"UnmarshalElement": strings.Join(setSliceElement, "\n"),
		"ElementSource":    elementSource,
		"ElementTarget":    elementTarget,
	}); err != nil {
		return nil, err
	}
	return strings.Split(buf.String(), "\n"), nil
}

var assignValueTmpl = template.Must(template.New("assignValueTMPL").Parse(
	`
	{{.Target}}Value, ok := {{.Source}}.(values.Array)
	if !ok {
		err = errors.Errorf("cannot cast %T to %s", {{.Source}}, reflect.TypeFor[*values.DeviceArray]().Name())
		return
	}
	{{.Target}} := types.New{{.GoType}}[{{.GoDataType}}]({{.Target}}Value)
`))

func (b *binder) processArrayValue(target, src string, dataType ir.Type, goType string) ([]string, error) {
	goDataType, err := b.nameGoType(dataType)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := assignValueTmpl.Execute(&buf, map[string]string{
		"Source":     src,
		"Target":     target,
		"GoType":     goType,
		"GoDataType": goDataType,
	}); err != nil {
		return nil, err
	}
	return []string{buf.String()}, nil
}

func (b *binder) setTargetFromSourceType(target, src string, typ ir.Type) ([]string, error) {
	switch typT := typ.(type) {
	case *ir.AtomicType:
		return b.processArrayValue(target, src, typT, "Atom")
	case *ir.ArrayType:
		return b.processArrayValue(target, src, typT.DataType(), "Array")
	case *ir.NamedType:
		return b.processNamedTypeOutput(target, src, typT)
	case *ir.SliceType:
		return b.processSliceOutput(target, src, typT)
	default:
		return nil, errors.Errorf("result type not supported: %T", typT)
	}
}

func (b *binder) namedTypeFactory(typ *ir.NamedType) (string, error) {
	fac := "Factory"
	if typ.Package() == b.Package {
		return fac, nil
	}
	return b.namePackage(typ.Package()) + "." + fac, nil
}

// Generate the bindings for Go.
func Generate(builder DependenciesBuildCall, w io.Writer, pkg *ir.Package) error {
	b := &binder{
		Package:      pkg,
		builder:      builder,
		dependencies: make(map[string]dependency),
		stdlib:       stdlib.Importer(nil),
	}
	b.funcs = buildFuncs(b)
	var err error
	b.NamedTypes, err = buildTypes(b)
	if err != nil {
		return err
	}

	b.FuncRunners, err = b.funcRunners(b.funcs)
	if err != nil {
		return err
	}
	b.NamedTypeDefinitions, err = b.namedTypeDefinitions()
	if err != nil {
		return err
	}
	b.PkgVars, err = b.buildPkgVars()
	if err != nil {
		return err
	}
	source := bytes.Buffer{}
	if err := goPackageTemplate.Execute(&source, b); err != nil {
		return err
	}
	formatted, err := format.Source(source.Bytes())
	if err != nil {
		// We copy the generated content to the writer for debugging.
		io.Copy(w, &source)
		return errors.Errorf("cannot format source code: %v", err)
	}
	_, err = w.Write(formatted)
	return err
}

func tmplExecute(tmpl *template.Template, data any) (string, error) {
	bld := strings.Builder{}
	if err := tmpl.Execute(&bld, data); err != nil {
		return "", err
	}
	return bld.String(), nil
}
