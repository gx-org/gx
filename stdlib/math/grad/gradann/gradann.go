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

// Package gradann provides an annotation to set the gradient of a function.
package gradann

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir/annotations"
	"github.com/gx-org/gx/build/ir"
)

// Annotation used to specify the gradient of a function.
type Annotation struct {
	partials []ir.PkgFunc
	vjp      ir.PkgFunc
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
			partials: make([]ir.PkgFunc, fType.Params.Len()),
		}
	})
}

// Partial returns the partial derivative for a given function parameter.
func (ann *Annotation) Partial(i int) ir.PkgFunc {
	return ann.partials[i]
}

// SetPartial returns the partial derivative for a given function parameter.
func (ann *Annotation) SetPartial(i int, fn ir.PkgFunc) error {
	ann.partials[i] = fn
	if ann.vjp != nil {
		return errors.Errorf("VJP already defined")
	}
	return nil
}

// SetVJP returns the partial derivative for a given function parameter.
func (ann *Annotation) SetVJP(fn ir.PkgFunc) error {
	for _, p := range ann.partials {
		if p != nil {
			return errors.Errorf("gradient funtion already defined")
		}
	}
	ann.vjp = fn
	return nil
}

// VJP returns a vjp function if it has been set.
// Returns nil if it has not been defined.
func (ann *Annotation) VJP() ir.PkgFunc {
	return ann.vjp
}
