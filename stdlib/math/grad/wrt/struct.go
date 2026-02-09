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

package wrt

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir"
)

// Struct is a parameter in a function with the struct type.
type Struct struct {
	*core
	typ      *ir.StructType
	children WRTs
}

func (cr *core) parseStructure(backwardValues *ast.FieldList, tp *ir.StructType) (WRT, error) {
	st := &Struct{
		core: cr,
		typ:  tp,
	}
	fields := tp.Fields.Fields()
	st.children = make([]WRT, len(fields))
	for i, field := range tp.Fields.Fields() {
		var err error
		st.children[i], err = parse(backwardValues, st, field)
		if err != nil {
			return nil, err
		}
	}
	return st, nil
}

// Arrays returns the list of array for which the gradient needs to be computed with respect to.
func (st *Struct) Arrays() []*Array {
	var arrays []*Array
	for _, child := range st.children {
		arrays = append(arrays, child.Arrays()...)
	}
	return arrays
}
