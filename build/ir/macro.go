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

package ir

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir/annotations"
)

// MacroCall calls a macro.
type MacroCall struct {
	X Expr // Expression to call to the macro
	F Func // Result from calling the macro
}

var (
	_ Expr = (*MacroCall)(nil)
	_ Func = (*MacroCall)(nil)
)

func (*MacroCall) node()         {}
func (*MacroCall) staticValue()  {}
func (*MacroCall) assignable()   {}
func (*MacroCall) storage()      {}
func (*MacroCall) storageValue() {}

// Name of the function. Returns an empty string since literals are always anonymous.
func (s *MacroCall) Name() string {
	return ""
}

// Store returns the function literal as a store.
func (s *MacroCall) Store() Storage {
	return s
}

// NameDef returns nil because function literals are anonymous.
func (s *MacroCall) NameDef() *ast.Ident {
	return nil
}

// Value returns the function literal as an assignable expression.
func (s *MacroCall) Value(x Expr) AssignableExpr {
	return &FuncValExpr{
		X: s,
		F: s.F,
		T: s.F.FuncType(),
	}
}

// ShortString returns the name of the function.
func (s *MacroCall) ShortString() string {
	return s.X.String()
}

// Doc returns associated documentation or nil.
func (s *MacroCall) Doc() *ast.CommentGroup {
	return nil
}

// File declaring the function literal.
func (s *MacroCall) File() *File {
	return s.F.File()
}

// Type returns the type of the function.
func (s *MacroCall) Type() Type {
	return s.F.Type()
}

// FuncType returns the concrete type of the function.
func (s *MacroCall) FuncType() *FuncType {
	return s.F.FuncType()
}

// Same returns true if the other storage is this storage.
func (s *MacroCall) Same(o Storage) bool {
	return Storage(s) == o
}

// Source returns the node in the AST tree.
func (s *MacroCall) Source() ast.Node { return s.X.Source() }

// Annotations returns the annotations attached to the function.
func (s *MacroCall) Annotations() *annotations.Annotations {
	return s.F.Annotations()
}

// String returns the expression calling the macro.
func (s *MacroCall) String() string {
	return s.X.String()
}
