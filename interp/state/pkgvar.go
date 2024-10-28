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
	"github.com/pkg/errors"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

// pkgVar is package (static) variable).
type pkgVar struct {
	state *State
	decl  *ir.VarDecl
	value *values.HostArray
	nods  []*graph.OutputNode
}

var (
	_ backendElement      = (*pkgVar)(nil)
	_ ElementWithConstant = (*pkgVar)(nil)
)

// PkgVar returns a new node representing a scalar value.
func (g *State) PkgVar(decl *ir.VarDecl, value values.Value) (NumericalElement, error) {
	hostValue, ok := value.(*values.HostArray)
	if !ok {
		return nil, errors.Errorf("cannot create a package variable for type %s: not supported", value.Type().String())
	}
	elmt := &pkgVar{
		decl:  decl,
		value: hostValue,
	}
	nod, err := g.backendGraph.Core().NewConstant(hostValue.Buffer())
	if err != nil {
		return nil, err
	}
	elmt.nods = []*graph.OutputNode{&graph.OutputNode{
		Node:  nod,
		Shape: hostValue.Shape(),
	}}
	return elmt, nil
}

func (n *pkgVar) Shape() *shape.Shape {
	return n.value.Shape()
}

func (n *pkgVar) NumericalConstant() *values.HostArray {
	return n.value
}

func (n *pkgVar) nodes() ([]*graph.OutputNode, error) {
	return n.nods, nil
}

// State owning the element.
func (n *pkgVar) State() *State {
	return n.state
}

// Type returns the type of the structure.
func (n *pkgVar) Type() ir.Type {
	return n.decl.TypeV
}

func (n *pkgVar) valueFromHandle(handles *handleParser) (values.Value, error) {
	return values.NewDeviceArray(n.Type(), handles.next()), nil
}

// ToExpr returns the value as a GX IR expression.
func (n *pkgVar) ToExpr() *HostArrayExpr {
	return &HostArrayExpr{
		Typ: n.decl.TypeV,
		Val: n.NumericalConstant(),
	}
}
