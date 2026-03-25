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
	"go/ast"

	"github.com/gx-org/gx/build/ir/irkind"
)

// nilType is the type returned by function with no results.
type nilType struct {
	distinctType
}

var nilTypeT = &nilType{distinctType: distinctType{kind: irkind.Nil}}

// NilType returns the type for the nil identifier.
func NilType() Type {
	return nilTypeT
}

// Nil implements the storage for the nil keyword.
type Nil struct {
	Src *ast.Ident
}

var _ Storage = (*Nil)(nil)

// NilStorage returns a nil storage given a nil identifier.
func NilStorage(src *ast.Ident) *Nil {
	return &Nil{Src: src}
}

func (*Nil) node()    {}
func (*Nil) storage() {}

// Node returns the AST node of the storage.
func (n *Nil) Node() ast.Node {
	return n.Src
}

// NameDef returns the identifier of the storage.
func (n *Nil) NameDef() *ast.Ident {
	return n.Src
}

// Same returns true if the other storage is a nil storage as well.
func (*Nil) Same(other Storage) bool {
	_, ok := other.(*Nil)
	return ok
}

// Type of the Nil storage.
func (*Nil) Type() Type {
	return NilType()
}
