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
	"fmt"
	"go/ast"

	"github.com/gx-org/gx/build/ir/irkind"
)

// Interface represents a set of types.
type Interface struct {
	BaseType[*ast.InterfaceType]
	types []Type
}

var (
	_ Type        = (*Interface)(nil)
	_ ArrayType   = (*Interface)(nil)
	_ assignsFrom = (*Interface)(nil)
)

// NewInterface returns a new type set given a set of types.
func NewInterface(src *ast.InterfaceType, types []Type) *Interface {
	if src == nil {
		src = &ast.InterfaceType{}
	}
	return &Interface{
		BaseType: BaseType[*ast.InterfaceType]{Src: src},
		types:    types,
	}
}

func (*Interface) node() {}

// Rank of the array.
func (*Interface) Rank() ArrayRank { return scalarRank }

// DataType returns the type of the element.
func (s *Interface) DataType() Type {
	return s
}

// ArrayType returns the source code defining the type.
// Always returns nil.
func (s *Interface) ArrayType() ast.Expr {
	return s.BaseType.Src
}

// Kind returns the scalar kind.
func (s *Interface) Kind() irkind.Kind { return irkind.Interface }

// Equal returns true if other is the exact same type set.
func (s *Interface) Equal(fetcher Fetcher, target Type) (bool, CompEvalError, error) {
	targetSet, ok := target.(*Interface)
	if !ok {
		return false, nil, nil
	}
	if s == targetSet {
		return true, nil, nil
	}
	if len(s.types) != len(targetSet.types) {
		return false, nil, nil
	}
	for i, typ := range s.types {
		ok, cpErr, err := typ.Equal(fetcher, targetSet.types[i])
		if err != nil {
			err = fmt.Errorf("cannot compare type set %s to %s: %w", s.ReferString(fetcher.File()), targetSet.ReferString(fetcher.File()), err)
			return false, nil, err
		}
		if cpErr != nil || !ok {
			return false, cpErr, nil
		}
	}
	return true, nil, nil
}

// AssignableTo reports whether a value of the type can be assigned to another.
func (s *Interface) AssignableTo(fetcher Fetcher, target Type) (bool, CompEvalError, error) {
	if targetSet, ok := target.(*Interface); ok {
		return targetSet.assignableFrom(fetcher, s)
	}
	for _, typ := range s.types {
		ok, cpErr, err := typ.AssignableTo(fetcher, target)
		if !ok || cpErr != nil || err != nil {
			return false, cpErr, err
		}
	}
	return len(s.types) > 0, nil, nil
}

// AssignableFrom reports whether a given source type is assignable to any members of the set.
func (s *Interface) assignableFrom(fetcher Fetcher, source Type) (bool, CompEvalError, error) {
	if len(s.types) == 0 {
		return true, nil, nil
	}

	if sourceSet, ok := source.(*Interface); ok {
		return s.containsTypes(fetcher, sourceSet)
	}
	for _, typ := range s.types {
		ok, cpErr, err := source.AssignableTo(fetcher, typ)
		if cpErr != nil || err != nil {
			return false, cpErr, err
		}
		if ok {
			return true, nil, nil
		}
	}
	return false, nil, nil
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *Interface) ConvertibleTo(fetcher Fetcher, target Type) (bool, CompEvalError, error) {
	if _, ok := target.(*Interface); ok {
		return s.Equal(fetcher, target)
	}
	for _, typ := range s.types {
		ok, cpErr, err := typ.ConvertibleTo(fetcher, target)
		if cpErr != nil || err != nil {
			return false, cpErr, err
		}
		if ok {
			return true, nil, nil
		}
	}
	return false, nil, nil
}

// Specialise a type to a given target.
func (s *Interface) Specialise(Specialiser) (Type, error) {
	return s, nil
}

// UnifyWith recursively unifies a type parameters with types.
func (*Interface) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

var anyType = &Interface{}

// AnyType returns the type for the keyword any.
func AnyType() Type {
	return anyType
}
