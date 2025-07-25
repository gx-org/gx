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

// Package ccbindings generates C++ bindings for a GX package.
package ccbindings

import (
	"io"
	"strings"
	"text/template"

	"github.com/gx-org/gx/golang/binder/ccbindings/fmtpath"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/binder/bindings"

	_ "embed"
)

var (
	//go:embed bindings.cc.tmpl
	ccBindings string
	ccTemplate = template.Must(template.New("ccTMPL").Parse(ccBindings))

	//go:embed bindings.h.tmpl
	headerBindings string
	hTemplate      = template.Must(template.New("hTMPL").Parse(headerBindings))
)

type binder struct {
	Package *ir.Package
	Class   string

	Funcs []*function
}

// New C++ binder.
func New(pkg *ir.Package) (bindings.Binder, error) {
	name := pkg.Name.Name
	b := &binder{
		Package: pkg,
		Class:   strings.ToUpper(name[:1]) + name[1:],
	}
	var err error
	b.Funcs, err = bindings.BuildFuncs(pkg, b.newFunc)
	if err != nil {
		return nil, err
	}
	return b, nil

}

func (b *binder) Files() []bindings.File {
	return []bindings.File{
		headerFile{binder: b},
		sourceFile{binder: b},
	}
}

func (b *binder) Namespace() string {
	return fmtpath.Namespace(b.Package.FullName())
}

type headerFile struct {
	*binder
}

func (headerFile) Extension() string {
	return ".h"
}

func (f headerFile) WriteBindings(w io.Writer) error {
	return hTemplate.Execute(w, f)
}

func (f headerFile) HeaderGuard() string {
	return fmtpath.HeaderGuard(f.Package.FullName())
}

type sourceFile struct {
	*binder
}

func (sourceFile) Extension() string {
	return ".cc"
}

func (f sourceFile) WriteBindings(w io.Writer) error {
	return ccTemplate.Execute(w, f)
}

func (f sourceFile) HeaderPath() string {
	return fmtpath.HeaderPath(f.Package.FullName())
}
