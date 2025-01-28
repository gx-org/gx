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
	"github.com/gx-org/gx/golang/binder/bindings"
	"github.com/gx-org/gx/stdlib"

	_ "embed"
)

//go:embed bindings.go.tmpl
var goBindings string

var goPackageTemplate = template.Must(template.New("GoBindingsTMPL").Parse(goBindings))

type (
	// DependenciesBuildCall generates the source code to get the GX package
	// in GX intermediate representation (IR).
	DependenciesBuildCall interface {
		// SourceImport generates the import when the GX source needs to be imported
		// from a Go module.
		SourceImport(*ir.Package) string

		// DependencyImport generates the import to insert in the package source code.
		DependencyImport(*ir.Package) string

		// StdlibDependencyImport returns the package path to a dependency on
		// the standard library package.
		StdlibDependencyImport(string) string
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

var processNamedTypeOutputTmpl = template.Must(template.New("processNamedTypeOutputTMPL").Parse(
	`var {{.Target}} {{.TypePointer}}
{{.Target}}, err = {{.Package}}.Marshal{{.TypeName}}({{.Source}})
if err != nil {
	return
}`))

func (b *binder) processNamedTypeOutput(target, src string, typ *ir.NamedType) (res []string, err error) {
	pkg := "cmpl"
	if typ.Package() != b.Package {
		pkg += "." + b.namePackage(typ.Package())
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
		"Package":     pkg,
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
	case ir.ArrayType:
		if typT.Rank().IsAtomic() {
			return b.processArrayValue(target, src, typT, "Atom")
		}
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

// New returns a new Go bindings generator.
func New(builder DependenciesBuildCall, pkg *ir.Package) (bindings.Binder, error) {
	b := &binder{
		Package:      pkg,
		builder:      builder,
		dependencies: make(map[string]dependency),
		stdlib:       stdlib.Importer(nil),
	}
	var err error
	b.funcs, err = bindings.BuildFuncs(pkg, b.newFunc)
	if err != nil {
		return nil, err
	}
	b.NamedTypes, err = buildTypes(b)
	if err != nil {
		return nil, err
	}

	b.FuncRunners, err = b.funcRunners(b.funcs)
	if err != nil {
		return nil, err
	}
	b.NamedTypeDefinitions, err = b.namedTypeDefinitions()
	if err != nil {
		return nil, err
	}
	b.PkgVars, err = b.buildPkgVars()
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Files returns the list of files to generate.
func (b *binder) Files() []bindings.File {
	return []bindings.File{b}
}

// Extension returns the extension of the file being generated.
func (b *binder) Extension() string {
	return ".go"
}

// WriteBindings the bindings for Go.
func (b *binder) WriteBindings(w io.Writer) error {
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
