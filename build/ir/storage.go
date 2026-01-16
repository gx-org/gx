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
	val  Expr
}

var builtinStore builtinStorage

func (*builtinStorage) node()         {}
func (*builtinStorage) storage()      {}
func (*builtinStorage) storageValue() {}

func (s *builtinStorage) Node() ast.Node {
	return s.name
}

func (s *builtinStorage) NameDef() *ast.Ident {
	return s.name
}

func (s *builtinStorage) Type() Type {
	return s.val.Type()
}

func (s *builtinStorage) Value(Expr) Expr {
	return s.val
}

func (s *builtinStorage) String() string {
	return s.val.String()
}

// Same returns true if the other storage is this storage.
func (s *builtinStorage) Same(o Storage) bool {
	return Storage(s) == o
}

const numGXBuiltins = 17

var builtins = make(map[Storage]bool, numGXBuiltins)

func registerLanguageBuiltins(stor StorageWithValue) StorageWithValue {
	if len(builtins) > numGXBuiltins {
		// Check that builtins are not registered multiple times.
		panic("too many GX builtins registered")
	}
	builtins[stor] = true
	return stor
}

// BuiltinStorage stores a value in a builtin language storage.
func BuiltinStorage(name string, val Expr) StorageWithValue {
	return registerLanguageBuiltins(&builtinStorage{
		name: &ast.Ident{Name: name},
		val:  val,
	})
}

// BuiltinFunction registers a language builtin function.
func BuiltinFunction(impl FuncImpl) StorageWithValue {
	return registerLanguageBuiltins(&FuncBuiltin{
		Src:  &ast.FuncDecl{Name: &ast.Ident{Name: impl.Name()}},
		Impl: impl,
	})
}

// IsBuiltin returns true if the storage is GX language builtin (e.g. true, float32, append, ...)
func IsBuiltin(stor Storage) bool {
	return builtins[stor]
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
