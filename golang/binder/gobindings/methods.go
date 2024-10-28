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
	"text/template"

	"github.com/gx-org/gx/build/ir"
)

type (
	receiver interface {
		named() *ir.NamedType
		Index() int
	}

	method struct {
		Receiver receiver
		Method   function

		NameDevice string
	}

	methods struct {
		binder  *binder
		methods []method
	}
)

func (b *binder) buildMethods(r receiver) *methods {
	var funcs []method
	for i, fun := range r.named().Methods {
		if !ir.IsExported(fun.Name()) {
			continue
		}
		if fun.Type() == nil {
			continue
		}
		if !b.canBeOnDeviceFunc(fun) {
			continue
		}
		funcs = append(funcs, method{
			Receiver: r,
			Method: function{
				binder:    b,
				Func:      fun,
				FuncIndex: i,
			},

			NameDevice: fun.Name(),
		})
	}
	return &methods{binder: b, methods: funcs}
}

func (ms methods) Methods() (string, error) {
	return iterateFunc(ms.methods, func(_ int, m method) (string, error) {
		return fmt.Sprintf("%s() *Method%s%s", m.Method.Name(), m.Receiver.named().Name(), m.Method.Name()), nil
	})
}

func (ms methods) funcs() []*function {
	funcs := make([]*function, len(ms.methods))
	for i, method := range ms.methods {
		funcs[i] = &method.Method
	}
	return funcs
}

func (ms methods) Runners() (string, error) {
	return ms.binder.funcRunners(ms.funcs())
}

var handleMethodFieldTemplate = template.Must(template.New("handleMethodFieldTMPL").Parse(`
	runner{{.Method.Name}} *{{.Method.RunnerType}}
`))

func (ms methods) HandleMethodFields() (string, error) {
	return iterateTmpl(ms.methods, handleMethodFieldTemplate)
}

var methodReturnHandleTemplate = template.Must(template.New("methodReturnHandleTMPL").Parse(`
// {{.Method.Name}} returns a handle to compile method {{.Method.Name}} for a device.
func (s {{.Receiver.Named.Name}}) {{.Method.Name}}() *{{.Method.RunnerType}} {
	return s.handle.runner{{.Method.Name}}
}
`))

func (ms methods) ReturnHandle() (string, error) {
	return iterateTmpl(ms.methods, methodReturnHandleTemplate)
}

var methodInitRunnerTemplate = template.Must(template.New("methodInitRunnerTMPL").Parse(`
	s.handle.runner{{.Method.Name}} = &{{.Method.RunnerType}}{
		methodBase: s.handle.compiler.{{.Method.RunnerField}},
		receiver: s.handle,
	}
`))

func (ms methods) InitRunners() (string, error) {
	return iterateTmpl(ms.methods, methodInitRunnerTemplate)
}

var methodCompilerSetField = template.Must(template.New("funcCompilerSetFieldTMPL").Parse(`
	c.{{.Method.RunnerField}} = methodBase{
		compiler: c,
		function: c.Package.IR.Types[{{.Receiver.Index}}].Methods[{{.Method.FuncIndex}}].(*ir.FuncDecl),
	}`))

func (b *binder) MethodsCompilerSetFields() (string, error) {
	return iterateFunc(b.NamedTypes, func(_ int, typ namedType) (string, error) {
		return iterateTmpl(typ.Methods().methods, methodCompilerSetField)
	})
}
