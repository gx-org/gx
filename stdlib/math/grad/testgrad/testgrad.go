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

// Package testgrad provides function to test autograd.
package testgrad

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
)

func listFunc(pkg *ir.Package) []string {
	fns := pkg.Decls.Funcs
	ss := make([]string, len(fns))
	for i, fn := range pkg.Decls.Funcs {
		ss[i] = fn.Name()
	}
	return ss
}

func checkFunc(pkg *ir.Package, name string, want string) error {
	if want == "" {
		return nil
	}
	names := strings.Split(name, ".")
	var gotF fmt.Stringer
	switch len(names) {
	case 1:
		gotF = pkg.FindFunc(name)
		if gotF == nil {
			return errors.Errorf("cannot find function %s. Available functions are %v", name, listFunc(pkg))
		}
	case 2:
		tpName := names[0]
		methodName := names[1]
		tp := pkg.Decls.TypeByName(tpName)
		if tp == nil {
			return errors.Errorf("type name %s not found", tpName)
		}
		gotF = tp.MethodByName(methodName)
		if gotF == nil {
			return errors.Errorf("method %s not found for type %s", methodName, tpName)
		}
	default:
		return errors.Errorf("cannot find %s: not supported", name)
	}
	return testbuild.CompareString(gotF.String(), want)
}
