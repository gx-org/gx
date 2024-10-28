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

package state

import (
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

// Zero represents a zero element.
type Zero struct {
	state *State
	typ   ir.Type
}

// NewZero returns an element representing a zero value given a GX type.
func (g *State) NewZero(typ ir.Type) *Zero {
	return &Zero{state: g, typ: typ}
}

// State owning the element.
func (n *Zero) State() *State {
	return n.state
}

func (n *Zero) nodes() ([]*graph.OutputNode, error) {
	return nil, nil
}

func (n *Zero) valueFromHandle(handles *handleParser) (values.Value, error) {
	return values.Zero(n.typ)
}
