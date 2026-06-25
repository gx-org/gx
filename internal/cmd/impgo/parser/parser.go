// Copyright 2026 Google LLC
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

// Package parser parses a Go package.
package parser

import (
	"go/types"

	"github.com/gx-org/gx/internal/cmd/impgo/generator"
)

// Visitor produces the code given what is found in the Go package.
type Visitor interface {
	Func(f *types.Func) error
	Struct(name *types.TypeName, s *types.Struct) error
	Interface(name *types.TypeName, s *types.Interface) error
}

func walkTypeName(v Visitor, tn *types.TypeName) error {
	under := tn.Type().Underlying()
	switch underT := under.(type) {
	case *types.Struct:
		return v.Struct(tn, underT)
	case *types.Interface:
		return v.Interface(tn, underT)
	}
	return nil
}

// Walk all the code required from a target.
func Walk(target generator.Target, v Visitor) error {
	pkg := target.Src.Pkg
	for _, name := range pkg.Scope().Names() {
		obj := pkg.Scope().Lookup(name)
		if !obj.Exported() {
			continue
		}
		switch objT := obj.(type) {
		case *types.Func:
			if err := v.Func(objT); err != nil {
				return err
			}
		case *types.TypeName:
			if err := walkTypeName(v, objT); err != nil {
				return err
			}
		}
	}
	return nil
}
