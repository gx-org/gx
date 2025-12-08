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

// Package setann provides an annotation to set the gradient of a function.
package setann

import (
	"github.com/gx-org/gx/build/ir/annotations"
	"github.com/gx-org/gx/build/ir"
)

// Annotation used to specify the gradient of a function.
type Annotation struct {
	Partials []ir.PkgFunc
}

var key = annotations.NewKey(Annotation{})

// Get returns the grad.Set annotation given a function.
// Returns nil if no annotation has been set.
func Get(f ir.Func) *Annotation {
	return annotations.Get[*Annotation](f, key)
}

// GetDef returns an annotation if one already exists or create a new one if it is not already set.
func GetDef(f ir.Func) *Annotation {
	fType := f.FuncType()
	return annotations.GetDef(f, key, func() *Annotation {
		return &Annotation{
			Partials: make([]ir.PkgFunc, fType.Params.Len()),
		}
	})
}
