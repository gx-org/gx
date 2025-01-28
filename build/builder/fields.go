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
	"iter"
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
	for i, astField := range f.ext.Src.List {
		var groupOk bool
		f.list[i], groupOk = processFieldGroup(owner, astField, assign)
		ok = ok && groupOk
	}
	return f, ok
}

func importFieldList(scope scoper, fields *ir.FieldList) (*fieldList, bool) {
	if fields == nil {
		return nil, true
	}
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

// resolveType recursively calls resolveType in the underlying subtree of nodes, and returns the
// results in a new fieldList.
func (f *fieldList) resolveType(scope scoper) (*fieldList, bool) {
	if f == nil {
		return nil, true
	}
	ok := true
	out := fieldList{
		ext:  f.ext,
		list: make([]*fieldGroup, len(f.list)),
	}
	for i, group := range f.list {
		var fieldOk bool
		out.list[i] = &fieldGroup{
			ext:  group.ext,
			list: make([]*field, len(group.list)),
		}
		for j, fld := range group.list {
			out.list[i].list[j] = &field{
				ext:   fld.ext,
				group: out.list[i],
			}
		}
		out.list[i].typ, fieldOk = resolveType(scope, group, group.typ)
		ok = fieldOk && ok
	}
	return &out, ok
}

func (f *fieldList) irType() *ir.FieldList {
	if f == nil {
		return nil
	}
	if f.ext.List != nil {
		return f.ext
	}
	f.ext.List = make([]*ir.FieldGroup, len(f.list))
	for i, group := range f.list {
		f.ext.List[i] = group.irType()
	}
	return f.ext
}

func (f *fieldList) empty() bool {
	return f == nil || len(f.list) == 0
}

func (f *fieldList) isGeneric() bool {
	for _, field := range f.fields() {
		if field.typ().isGeneric() {
			return true
		}
	}
	return false
}

// fieldsSlice returns a newly-allocated slice containing all fields in the fieldList.
func (f *fieldList) fieldsSlice() []*field {
	var flds []*field
	for _, field := range f.fields() {
		flds = append(flds, field)
	}
	return flds
}

// fields returns an iterator over all fields in the fieldList. The iterator returns the index
// of the field in the fieldList and the field itself, and can be used with range.
func (f *fieldList) fields() iter.Seq2[int, *field] {
	return func(yield func(int, *field) bool) {
		if f == nil {
			return
		}
		n := 0
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
			for _, field := range fields {
				if !yield(n, field) {
					return
				}
				n++
			}
		}
	}
}

func (f *fieldList) numFields() int {
	num := 0
	if f != nil {
		for _, field := range f.list {
			if field.ext.Src != nil && field.ext.Src.Names != nil {
				num += len(field.ext.Src.Names)
				continue
			}
			num++
		}
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

func processFieldGroup(owner owner, src *ast.Field, assign func(*scopeFile, *field) bool) (*fieldGroup, bool) {
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
			},
		}
		assignOk := assign(owner.fileScope(), field)
		ok = ok && assignOk
		grp.ext.Fields[i] = field.ext
		grp.list = append(grp.list, field)
	}
	return grp, ok
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

func (f *fieldGroup) irType() *ir.FieldGroup {
	f.ext.Type = f.typ.irType()
	return f.ext
}

func (f *fieldGroup) source() ast.Node {
	return f.ext.Source()
}

func (f *fieldGroup) String() string {
	if len(f.ext.Src.Names) == 0 {
		return f.typ.String()
	}
	names := make([]string, len(f.ext.Src.Names))
	for i, name := range f.ext.Src.Names {
		names[i] = name.Name
	}
	return strings.Join(names, ",") + " " + f.typ.String()
}

func toIRTypes(tps []typeNode) []ir.Type {
	irs := make([]ir.Type, len(tps))
	for i, typ := range tps {
		irs[i] = typ.irType()
	}
	return irs
}

func (f *field) typ() typeNode {
	return f.group.typ
}
