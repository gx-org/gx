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

package grad

import (
	"go/ast"

	"github.com/gx-org/gx/base/uname"
)

type namedFields struct {
	fields *ast.FieldList
	names  []string
}

func nameFields(unames *uname.Unique, root string, fields *ast.FieldList) *namedFields {
	all := ast.FieldList{
		List: make([]*ast.Field, len(fields.List)),
	}
	var names []string
	for iField, field := range fields.List {
		named := *field
		if len(named.Names) == 0 {
			named.Names = []*ast.Ident{&ast.Ident{}}
		}
		for iName, name := range named.Names {
			retName := uname.DefaultIdent(name, root)
			named.Names[iName] = unames.Ident(retName)
			names = append(names, retName.Name)
		}
		all.List[iField] = &named
	}
	return &namedFields{fields: &all, names: names}
}
