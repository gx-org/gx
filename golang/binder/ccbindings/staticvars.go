// Copyright 2025 Google LLC
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

package ccbindings

import "github.com/gx-org/gx/build/ir"

type staticVar struct {
	binder *binder
	*ir.VarExpr
}

func (b *binder) buildStaticVars(pkg *ir.Package) []*staticVar {
	var vars []*staticVar
	for _, vrSpec := range pkg.Decls.Vars {
		for _, vrExpr := range vrSpec.Exprs {
			if !ir.IsExported(vrExpr.NameDef().Name) {
				continue
			}
			vars = append(vars, &staticVar{
				binder:  b,
				VarExpr: vrExpr,
			})
		}
	}
	return vars
}

func (v *staticVar) Name() string {
	return v.NameDef().Name
}

func (v *staticVar) PkgPath() string {
	return v.VarExpr.Decl.FFile.Package.FullName()
}

func (v *staticVar) CCType() (string, error) {
	return v.binder.ccTypeFromKind(v.Type().Kind())
}
