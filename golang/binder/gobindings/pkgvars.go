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
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/base/tmpl"
	"github.com/gx-org/gx/build/ir"
)

var pkgVarsTmpl = template.Must(template.New("pkgVarsTMPL").Parse(`
var {{.VName.Name}} {{.VName.Name}}Static

type {{.VName.Name}}Static struct {
	value {{.GoType}}
}

func ({{.VName.Name}}Static) Set(value {{.GoType}}) interp.PackageOptionFactory {
	return func(plat platform.Platform) interp.PackageOption {
		hostValue := {{.ToHostValue}}
		return interp.PackageVarSetValue{
			Pkg: "{{.PackagePath}}",
			Var: "{{.VName.Name}}",
			Value: hostValue.GXValue(),
		}
	}
}
`))

type pkgVar struct {
	*binder
	*ir.VarDecl
	VarIndex int
}

func (v pkgVar) ToHostValue() (string, error) {
	kind := v.TypeV.Kind()
	if kind == ir.IntLenKind || kind == ir.IntIdxKind {
		return "types.DefaultInt(value)", nil
	}
	if ir.SupportOperators(kind) {
		return fmt.Sprintf("types.%s(value)", strings.Title(kind.String())), nil
	}
	return "", errors.Errorf("static variable of type %s not supported", kind.String())
}

func (v pkgVar) GoType() (string, error) {
	return v.nameGoType(v.TypeV)
}

func (v pkgVar) PackagePath() string {
	return v.Package.FullName()
}

func (b *binder) buildPkgVars() (string, error) {
	return tmpl.IterateFunc(b.Package.Vars, func(index int, decl *ir.VarDecl) (string, error) {
		buf := strings.Builder{}
		if err := pkgVarsTmpl.Execute(&buf, &pkgVar{
			binder:   b,
			VarDecl:  decl,
			VarIndex: index,
		}); err != nil {
			return "", errors.Errorf("cannot generate %s static variable declaration: %v", decl.VName.Name, err)
		}
		return buf.String(), nil
	})
}
