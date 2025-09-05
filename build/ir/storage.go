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
)

type builtinStorage struct {
	name *ast.Ident
	val  AssignableExpr
}

var builtinStore builtinStorage

func (*builtinStorage) node()         {}
func (*builtinStorage) storage()      {}
func (*builtinStorage) storageValue() {}

func (s *builtinStorage) Source() ast.Node {
	return s.name
}

func (s *builtinStorage) NameDef() *ast.Ident {
	return s.name
}

func (s *builtinStorage) Type() Type {
	return s.val.Type()
}

func (s *builtinStorage) Value(Expr) AssignableExpr {
	return s.val
}

func (s *builtinStorage) String() string {
	return s.val.String()
}

// Same returns true if the other storage is this storage.
func (s *builtinStorage) Same(o Storage) bool {
	return Storage(s) == o
}

// BuiltinStorage stores a value in a builtin storage.
func BuiltinStorage(name string, val AssignableExpr) StorageWithValue {
	return &builtinStorage{
		name: &ast.Ident{Name: name},
		val:  val,
	}
}

var (
	falseStorage = BuiltinStorage("false", False())
	trueStorage  = BuiltinStorage("true", True())
)

// FalseStorage returns the storage for the value of true.
func FalseStorage() StorageWithValue {
	return falseStorage
}

// TrueStorage returns the storage for the value of true.
func TrueStorage() StorageWithValue {
	return trueStorage
}
