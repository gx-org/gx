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

type (
	groupCloner func(*FieldGroup) (*FieldGroup, error)
	fieldCloner func(*FieldGroup, int, *Field) (*Field, error)

	cloner struct {
		group groupCloner
		field fieldCloner
	}
)

func cloneFields(list *FieldList, cl *cloner) (*FieldList, error) {
	if cl == nil || list == nil {
		return list, nil
	}
	ext := &FieldList{Src: list.Src}
	for _, group := range list.List {
		cloneGroup, err := cl.group(group)
		if err != nil {
			return nil, err
		}
		for i, field := range group.Fields {
			cloneField, err := cl.field(cloneGroup, i, field)
			if err != nil {
				return nil, err
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
	return ext, nil
}

func cloneGroup(grp *FieldGroup) (*FieldGroup, error) {
	return &FieldGroup{Src: grp.Src, Type: grp.Type}, nil
}

func cloneField(grp *FieldGroup, _ int, field *Field) (*Field, error) {
	return &Field{Name: field.Name, Group: grp}, nil
}
