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
	groupCloner func(*FieldGroup) (*FieldGroup, CompEvalError, error)
	fieldCloner func(*FieldGroup, int, *Field) (*Field, CompEvalError, error)

	cloner struct {
		group groupCloner
		field fieldCloner
	}
)

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
		for nameI, name := range grp.Fields {
			names[nameI] = name.Name.Name
		}
		groups[grpI] = fmt.Sprintf("%s %s", strings.Join(names, ", "), typeS)
	}
	return strings.Join(groups, ", ")
}

func cloneFields(list *FieldList, cl *cloner) (*FieldList, CompEvalError, error) {
	if cl == nil || list == nil {
		return list, nil, nil
	}
	ext := &FieldList{Src: list.Src}
	for _, group := range list.List {
		cloneGroup, cpErr, err := cl.group(group)
		if cpErr != nil || err != nil {
			return nil, cpErr, err
		}
		for i, field := range group.Fields {
			cloneField, cpErr, err := cl.field(cloneGroup, i, field)
			if err != nil {
				return nil, cpErr, err
			}
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
	return ext, nil, nil
}

func cloneGroup(grp *FieldGroup) (*FieldGroup, CompEvalError, error) {
	return &FieldGroup{Src: grp.Src, Type: grp.Type}, nil, nil
}

func cloneField(grp *FieldGroup, _ int, field *Field) (*Field, CompEvalError, error) {
	return &Field{Name: field.Name, Group: grp}, nil, nil
}
