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

package builder

import (
	"go/ast"
	"strings"

	"github.com/gx-org/gx/build/ir"
)

type fieldList struct {
	src  *ast.FieldList
	list []*fieldGroup
}

func processFieldList(pscope typeProcScope, src *ast.FieldList, assign func(procScope, *field) bool) (*fieldList, bool) {
	if src == nil {
		return nil, true // No fields.
	}
	f := &fieldList{src: src}
	f.list = make([]*fieldGroup, len(src.List))
	ok := true
	for i, astField := range src.List {
		var groupOk bool
		f.list[i], groupOk = processFieldGroup(pscope, astField, assign)
		ok = ok && groupOk
	}
	return f, ok
}

func (f *fieldList) buildFieldList(dscope *defineLocalScope) (*ir.FieldList, bool) {
	if f == nil {
		return nil, true
	}
	fList := &ir.FieldList{
		Src:  f.src,
		List: make([]*ir.FieldGroup, len(f.list)),
	}
	ok := true
	for i, group := range f.list {
		var fieldOk bool
		fList.List[i], fieldOk = group.build(dscope)
		ok = fieldOk && ok
	}
	if !ok {
		return fList, false
	}
	ok = buildTags(dscope, f, fList)
	return fList, ok
}

func (f *fieldList) empty() bool {
	return f == nil || len(f.list) == 0
}

func (f *fieldList) numFields() int {
	num := 0
	if f == nil {
		return num
	}
	for _, field := range f.list {
		if field.src != nil && field.src.Names != nil {
			num += len(field.src.Names)
			continue
		}
		num++
	}
	return num
}

func (f *fieldList) String() string {
	if f == nil {
		return ""
	}
	s := make([]string, len(f.list))
	for i, field := range f.list {
		s[i] = field.String()
	}
	return strings.Join(s, ",")
}

type fieldGroup struct {
	src  *ast.Field
	list []*field
	typ  typeExprNode
	tag  *tag
}

func processFieldGroup(pscope typeProcScope, src *ast.Field, assign func(procScope, *field) bool) (*fieldGroup, bool) {
	typ, ok := processTypeExpr(pscope, src.Type)
	grp := &fieldGroup{
		src:  src,
		list: make([]*field, len(src.Names)),
		typ:  typ,
	}
	for i, ident := range src.Names {
		field := processField(grp, ident)
		assignOk := assign(pscope, field)
		ok = ok && assignOk
		grp.list[i] = field
	}
	var tagOk bool
	grp.tag, tagOk = processTag(pscope, src.Tag)
	return grp, ok && tagOk
}

func (f *fieldGroup) build(dscope *defineLocalScope) (*ir.FieldGroup, bool) {
	grp := &ir.FieldGroup{
		Src:    f.src,
		Fields: make([]*ir.Field, len(f.list)),
	}
	var ok bool
	grp.Type, ok = f.typ.buildTypeExpr(dscope)
	for i, field := range f.list {
		grp.Fields[i] = field.build(dscope, grp)
	}
	return grp, ok
}

func (f *fieldGroup) String() string {
	if len(f.list) == 0 {
		return f.typ.String()
	}
	names := make([]string, len(f.list))
	for i := range len(names) {
		if f.src == nil {
			names[i] = "_"
		} else {
			names[i] = f.src.Names[i].Name
		}
	}
	return strings.Join(names, ",") + " " + f.typ.String()
}

type field struct {
	src   *ast.Ident
	group *fieldGroup
}

func processField(group *fieldGroup, src *ast.Ident) *field {
	return &field{src: src, group: group}
}

func (f *field) build(dscope *defineLocalScope, grp *ir.FieldGroup) *ir.Field {
	field := &ir.Field{Group: grp, Name: f.src}
	dscope.define(field.Storage())
	return field
}
