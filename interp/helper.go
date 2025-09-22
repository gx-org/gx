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

package interp

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/fun"
)

// PkgFuncFromElement extracts a function declaration from an element.
func PkgFuncFromElement(el ir.Element) (ir.PkgFunc, error) {
	fn, err := FuncFromElement(el)
	if err != nil {
		return nil, err
	}
	pkgFn, ok := fn.(ir.PkgFunc)
	if !ok {
		return nil, errors.Errorf("cannot convert element %T to a package function", el)
	}
	return pkgFn, nil
}

// FuncFromElement extracts a function declaration from an element.
func FuncFromElement(el ir.Element) (ir.Func, error) {
	fEl, ok := el.(fun.Func)
	if !ok {
		return nil, errors.Errorf("cannot convert element %T to a function", el)
	}
	return fEl.Func(), nil
}

// FuncDeclFromElement extracts a function declaration from an element.
func FuncDeclFromElement(el ir.Element) (*ir.FuncDecl, error) {
	fun, err := FuncFromElement(el)
	if err != nil {
		return nil, err
	}
	fDecl, ok := fun.(*ir.FuncDecl)
	if !ok {
		return nil, errors.Errorf("%s is not a GX user function", fun.Name())
	}
	return fDecl, nil
}
