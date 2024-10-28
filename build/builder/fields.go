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
	ext  *ir.FieldList
	list []*fieldGroup
}

func processFieldList(owner owner, expr *ast.FieldList, assign func(*scopeFile, *field) bool) (*fieldList, bool) {
	if expr == nil {
		return nil, true // No fields.
	}
	f := &fieldList{
		ext: &ir.FieldList{
			Src: expr,
		},
	}
	f.list = make([]*fieldGroup, len(expr.List))
	ok := true
	nextID := 0
	for i, astField := range f.ext.Src.List {
		var groupOk bool
		f.list[i], nextID, groupOk = processFieldGroup(owner, astField, assign, nextID)
		ok = ok && groupOk
	}
	return f, ok
}

func importFieldList(scope scoper, fields *ir.FieldList) (*fieldList, bool) {
	f := &fieldList{
		ext:  fields,
		list: make([]*fieldGroup, len(fields.List)),
	}
	ok := true
	for i, group := range fields.List {
		var fieldOk bool
		f.list[i], fieldOk = importFieldGroup(scope, group)
		ok = fieldOk && ok
	}
	return f, ok
}

func (f *fieldList) resolveType(scope scoper) (*fieldList, bool) {
	ok := true
	out := fieldList{
		ext:  f.ext,
		list: make([]*fieldGroup, len(f.list)),
	}
	for i, group := range f.list {
		var fieldOk bool
		out.list[i] = &fieldGroup{
			ext:  group.ext,
			list: group.list,
		}
		out.list[i].typ, fieldOk = resolveType(scope, group, group.typ)
		ok = fieldOk && ok
	}
	return &out, ok
}

func (f *fieldList) buildType() *ir.FieldList {
	if f == nil {
		return nil
	}
	if f.ext.List != nil {
		return f.ext
	}
	f.ext.List = make([]*ir.FieldGroup, len(f.list))
	for i, group := range f.list {
		f.ext.List[i] = group.buildType()
	}
	return f.ext
}

func (f *fieldList) fields() []*field {
	var flds []*field
	for _, group := range f.list {
		fields := group.list
		if len(fields) == 0 {
			fields = []*field{&field{
				ext: &ir.Field{
					Group: group.ext,
				},
				group: group,
			}}
		}
		flds = append(flds, fields...)
	}
	return flds
}

func (f *fieldList) numFields() int {
	num := 0
	for _, field := range f.list {
		if field.ext.Src != nil && field.ext.Src.Names != nil {
			num += len(field.ext.Src.Names)
			continue
		}
		num++
	}
	return num
}

func (f *fieldList) String() string {
	s := make([]string, len(f.list))
	for i, field := range f.list {
		s[i] = field.String()
	}
	return strings.Join(s, ",")
}

type (
	fieldGroup struct {
		ext  *ir.FieldGroup
		list []*field
		typ  typeNode
	}

	field struct {
		ext   *ir.Field
		group *fieldGroup
	}
)

func processFieldGroup(owner owner, src *ast.Field, assign func(*scopeFile, *field) bool, nextID int) (*fieldGroup, int, bool) {
	typ, ok := processTypeExpr(owner, src.Type)
	grp := &fieldGroup{
		ext: &ir.FieldGroup{
			Src:    src,
			Fields: make([]*ir.Field, len(src.Names)),
		},
		typ: typ,
	}
	for i, ident := range src.Names {
		field := &field{
			group: grp,
			ext: &ir.Field{
				Name:  ident,
				Group: grp.ext,
				ID:    nextID,
			},
		}
		assignOk := assign(owner.block(), field)
		if assignOk {
			nextID++
		} else {
			field.ext.ID = -1
		}
		ok = ok && assignOk
		grp.ext.Fields[i] = field.ext
		grp.list = append(grp.list, field)
	}
	return grp, nextID, ok
}

func importFieldGroup(scope scoper, irGroup *ir.FieldGroup) (*fieldGroup, bool) {
	grp := &fieldGroup{ext: irGroup}
	var ok bool
	grp.typ, ok = toTypeNode(scope, irGroup.Type)
	grp.list = make([]*field, len(irGroup.Fields))
	for i, irField := range irGroup.Fields {
		grp.list[i] = &field{ext: irField, group: grp}
	}
	return grp, ok
}

func (f *fieldGroup) buildType() *ir.FieldGroup {
	f.ext.Type = f.typ.buildType()
	return f.ext
}

func (f *fieldGroup) source() ast.Node {
	return f.ext.Source()
}

func (f *fieldGroup) String() string {
	names := make([]string, len(f.ext.Src.Names))
	for i, name := range f.ext.Src.Names {
		names[i] = name.Name
	}
	return strings.Join(names, ",") + " " + f.typ.String()
}

func toIRTypes(tps []typeNode) []ir.Type {
	irs := make([]ir.Type, len(tps))
	for i, typ := range tps {
		irs[i] = typ.buildType()
	}
	return irs
}

func (f *field) typ() typeNode {
	return f.group.typ
}
