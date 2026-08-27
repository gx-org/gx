// Copyright 2026 Google LLC
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

package surrogates

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/interp/elements"
)

type sStruct struct {
	core
	*elements.Struct
}

func newStruct(path storepath.Path, typ *ir.StructType) (Element, error) {
	fields := make(map[string]ir.Element)
	for _, field := range typ.Fields.Fields() {
		if !ir.ValidIdent(field.Name) {
			continue
		}
		fieldPath := storepath.NewSelect(path, field)
		var err error
		fields[field.Name.Name], err = New(fieldPath, field.Type())
		if err != nil {
			return nil, err
		}
	}
	return &sStruct{
		core:   core{path: path},
		Struct: elements.NewStruct(typ, fields),
	}, nil
}
