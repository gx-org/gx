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
	"strings"

	"github.com/gx-org/gx/build/ir/annotations"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

type labels struct {
	labels []string
}

var labelKey = annotations.NewKey(labels{})

func (l *labels) String() string {
	return strings.Join(l.labels, ",")
}

// Label a structure field with a gradient label.
func Label(fetcher ir.Fetcher, annotator *ir.AnnotatorField, field *ir.FieldGroup, call *ir.FuncCallExpr, args []ir.Element) (ir.FieldListCheckImpl, bool) {
	lbls := annotations.GetDef(field, labelKey, func() *labels {
		return &labels{}
	})
	newLabel, err := elements.StringFromElement(args[0])
	if err != nil {
		return nil, fetcher.Err().AppendAt(call.Args[0].Node(), err)
	}
	found := make(map[string]bool)
	for _, lbl := range lbls.labels {
		found[lbl] = true
	}
	if found[newLabel] {
		return nil, fetcher.Err().Appendf(call.Src, "field %q already has label %s", field.Fields[0].Name.Name, newLabel)
	}

	lbls.labels = append(lbls.labels, newLabel)
	return nil, true
}
