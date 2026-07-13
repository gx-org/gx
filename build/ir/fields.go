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

package ir

import (
	"fmt"
	"strings"
)

type (
	groupCloner func(*FieldGroup) *FieldGroup
	fieldCloner func(*FieldGroup, *Field) *Field

	cloner struct {
		group groupCloner
		field fieldCloner
	}
)

// ShortString represents the field list for an error message.
func (l *FieldList) ShortString(from *File) string {
	fields := l.Fields()
	types := make([]string, len(fields))
	for i, field := range fields {
		types[i] = field.Type().ReferString(from)
	}
	return strings.Join(types, ", ")
}

// SourceString represents the field list as GX source code.
func (l *FieldList) SourceString(from *File) string {
	groups := make([]string, len(l.List))
	for grpI, grp := range l.List {
		typeS := grp.Type.Val().ReferString(from)
		if len(grp.Fields) == 0 {
			groups[grpI] = typeS
			continue
		}
		names := make([]string, len(grp.Fields))
		for fieldI, field := range grp.Fields {
			name := "_"
			if field.Name != nil {
				name = field.Name.Name
			}
			names[fieldI] = name
		}
		groups[grpI] = fmt.Sprintf("%s %s", strings.Join(names, ", "), typeS)
	}
	return strings.Join(groups, ", ")
}

func cloneFields(list *FieldList, cl *cloner) *FieldList {
	if cl == nil || list == nil {
		return list
	}
	ext := &FieldList{Src: list.Src}
	for _, group := range list.List {
		cloneGroup := cl.group(group)
		for _, field := range group.Fields {
			cloneField := cl.field(cloneGroup, field)
			if cloneField == nil {
				continue
			}
			cloneField.Group = cloneGroup
			cloneGroup.Fields = append(cloneGroup.Fields, cloneField)
		}
		if len(cloneGroup.Fields) == 0 && len(group.Fields) > 0 {
			continue
		}
		ext.List = append(ext.List, cloneGroup)
	}
	return ext
}

func cloneGroup(grp *FieldGroup) *FieldGroup {
	res := *grp
	res.Fields = nil
	return &res
}

func cloneField(grp *FieldGroup, field *Field) *Field {
	res := *field
	if res.Orig == nil {
		res.Orig = field
	}
	res.Group = grp
	return &res
}
