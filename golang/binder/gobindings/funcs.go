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

	"github.com/gx-org/gx/base/tmpl"
	"github.com/gx-org/gx/build/ir"
)

type function struct {
	*binder
	ir.Func
	FuncIndex int
}

func (b *binder) newFunc(f ir.Func, i int) (*function, error) {
	return &function{
		binder:    b,
		Func:      f,
		FuncIndex: i,
	}, nil
}

func (b *binder) methods() []*function {
	funcs := []*function{}
	for _, namedType := range b.NamedTypes {
		funcs = append(funcs, namedType.Methods().funcs()...)
	}
	return funcs
}

func (b *binder) functionsAndMethods() []*function {
	return append(b.methods(), b.funcs...)
}

func (b *binder) FuncsPackageFields() (string, error) {
	return tmpl.IterateFunc(b.funcs, func(_ int, f *function) (string, error) {
		fieldName := f.RunnerField()
		fieldType := f.RunnerType()
		return fmt.Sprintf("\t%s %s", fieldName, fieldType), nil
	})
}

func (b *binder) MethodsPackageFields() (string, error) {
	return tmpl.IterateFunc(b.methods(), func(_ int, f *function) (string, error) {
		fieldName := f.RunnerField()
		const fieldType = "methodBase"
		return fmt.Sprintf("\t%s %s", fieldName, fieldType), nil
	})
}

var funcPackageSetField = template.Must(template.New("funcPackageSetFieldTMPL").Parse(`
	c.{{.RunnerField}} = {{.Name}}{
		methodBase: methodBase{
			pkg: c,
			function: c.Package.IR.Funcs[{{.FuncIndex}}].(*ir.FuncDecl),
		},
	}`))

func (b *binder) FuncsPackageSetFields() (string, error) {
	return tmpl.IterateTmpl(b.funcs, funcPackageSetField)
}

func (f function) RunnerField() string {
	fieldName := f.Name()
	if receiver := f.FuncType().Receiver; receiver != nil {
		fieldName = "method" + receiver.NameT + fieldName
	}
	return fieldName
}

func (f function) RunnerType() string {
	runnerType := f.Name()
	if receiver := f.FuncType().Receiver; receiver != nil {
		runnerType = "Method" + receiver.NameT + runnerType
	}
	return runnerType
}

func (f function) ReceiverField() string {
	receiver := f.FuncType().Receiver
	if receiver == nil {
		return ""
	}
	return fmt.Sprintf("receiver handle%s", receiver.NameT)
}

var funcStructTmpl = template.Must(template.New("funcStructTMPL").Parse(`
// {{.RunnerType}} compiles and runs the GX function {{.Name}} for a device.
{{with .Doc}}{{range .List -}}
{{.Text}}
{{end}}{{end -}}
type {{.RunnerType}} struct {
	methodBase
	{{.ReceiverField}}
}
`))

func (b *binder) funcRunners(funcs []*function) (string, error) {
	structS, err := tmpl.IterateTmpl(funcs, funcStructTmpl)
	if err != nil {
		return "", err
	}
	compileRunS, err := tmpl.IterateTmpl(funcs, funcRunnerTmpl)
	if err != nil {
		return "", err
	}
	return structS + compileRunS, nil
}

var funcRunnerTmpl = template.Must(template.New("funcRunnerTMPL").Parse(`
// Run first compiles {{.Name}} for a given device and the given arguments.
// Once compiled, the function is then run with these same arguments.
// If the shape of the arguments change, the function will panic.
func (f *{{.RunnerType}}) Run({{.Parameters}}) ({{.Results}}, err error) {
	var args []values.Value = {{.BackendArguments}}
	if f.runner == nil {
		f.runner, err = interp.Compile(f.pkg.Device, f.function.(*ir.FuncDecl), {{.ReceiverValue}}, args, f.pkg.options)
		if err != nil {
			return
		}
	}
	var outputs []values.Value
	outputs, err = f.runner.Run({{.ReceiverValue}}, args, f.pkg.Package.Tracer)
	if err != nil {
		return
	}

	{{.DefinePackageVariable}}{{.ProcessDeviceOutput}}
	return {{.Returns}}, nil
}
`))

func outputN(i int) string {
	return fmt.Sprintf("out%d", i)
}

func argN(i int) string {
	return fmt.Sprintf("arg%d", i)
}

func toKindSuffix(typ ir.Type) string {
	kind := typ.Kind()
	if kind == ir.IntLenKind || kind == ir.IntIdxKind {
		return "DefaultInt" // Matches platform.Host.DefaultInt(ir.Int)
	}
	kindS := kind.String()
	kindS = strings.ToUpper(kindS[:1]) + kindS[1:]
	return kindS
}

func (f function) ReceiverValue() string {
	receiver := f.Func.FuncType().Receiver
	if receiver == nil {
		return "nil"
	}
	return "f.receiver.GXValue()"
}

func (f function) DefinePackageVariable() string {
	for _, result := range f.Func.FuncType().Results.Fields() {
		if result.Type().Kind() == ir.StructKind {
			return strings.Join([]string{
				"cmpl := f.pkg",
			}, "\n\t")
		}
	}
	return ""
}

func (f function) ProcessDeviceOutput() (string, error) {
	fields := f.Func.FuncType().Results.Fields()
	lines := []string{""}
	for i, field := range fields {
		fieldLines, err := f.setTargetFromSourceType(outputN(i), fmt.Sprintf("outputs[%d]", i), field.Type())
		if err != nil {
			return "", err
		}
		lines = slices.Concat(lines, fieldLines)
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n\t"), nil
}

func (f function) BackendArguments() (string, error) {
	fields := f.Func.FuncType().Params.Fields()
	if len(fields) == 0 {
		return "nil", nil
	}
	var args = []string{
		"[]values.Value{",
	}
	for i, field := range fields {
		gxValue := fmt.Sprintf("%s.Bridge().GXValue()", argN(i))
		args = append(args,
			fmt.Sprintf("\t\t%s, // %s %s", gxValue, field.Name, field.Type()),
		)
	}
	args = append(args, "\t}")
	return strings.Join(args, "\n"), nil

}

func (f function) Parameters() (string, error) {
	fields := f.Func.FuncType().Params.Fields()
	params := make([]string, len(fields))
	for i, field := range fields {
		typeS, err := f.bridgerType(field.Group.Type)
		if err != nil {
			return "", err
		}
		params[i] = argN(i) + " " + typeS
	}
	return strings.Join(params, ", "), nil
}

func (f function) Returns() (string, error) {
	fields := f.Func.FuncType().Results.Fields()
	returns := make([]string, len(fields))
	for i := range fields {
		returns[i] = outputN(i)
	}
	return strings.Join(returns, ", "), nil
}

func (f function) Results() (string, error) {
	fields := f.Func.FuncType().Results.Fields()
	results := make([]string, len(fields))
	for i, field := range fields {
		typeS, err := f.binder.bridgerType(field.Group.Type)
		if err != nil {
			return "", err
		}
		results[i] = "_ " + typeS
	}
	return strings.Join(results, ", "), nil
}
