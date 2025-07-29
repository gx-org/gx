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

package ccbindings_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/golang/binder/bindings"
	"github.com/gx-org/gx/golang/binder/ccbindings"
	"github.com/gx-org/gx/golang/binder/ccbindings/fmtpath"
)

type loader struct{}

func (loader) Load(bld *builder.Builder, path string) (builder.Package, error) {
	paths := strings.Split(path, "/")
	if len(paths) != 1 {
		return nil, errors.Errorf("cannot load path %q: not supported", path)
	}
	pkg := bld.NewIncrementalPackage(path)
	if err := pkg.Build(fmt.Sprintf(`
package %s

func Fun__%s__() float32 {
	return 0
}
`, path, path)); err != nil {
		return nil, err
	}
	return pkg, nil
}

func compare(genFile bindings.File, want []string) error {
	var out strings.Builder
	if err := genFile.WriteBindings(&out); err != nil {
		return err
	}
	got := out.String()
	for _, expr := range want {
		if !strings.Contains(got, expr) {
			return errors.Errorf("generate source:\n%s\ndoes not contains expression %s", got, expr)
		}
	}
	return nil
}

func TestCCBindingsLocal(t *testing.T) {
	tests := []struct {
		pkgName    string
		wantHeader []string
		wantCC     []string
	}{
		{
			pkgName: "nopath",
			wantHeader: []string{
				"#ifndef NOPATH_H",
				"namespace nopath",
				"class Nopath;",
			},
			wantCC: []string{
				`#include "nopath.h"`,
				"namespace nopath",
				"Fun__nopath__Func",
			},
		},
	}
	for _, test := range tests {
		bld := builder.New(loader{})
		fmtpath.SetModuleName(test.pkgName)
		pkg, err := bld.Build(test.pkgName)
		if err != nil {
			t.Fatalf("cannot build package %s: %v", test.pkgName, err)
		}
		bnd, err := ccbindings.NewWithFmtPath(fmtpath.Functions{}, pkg.IR())
		if err != nil {
			t.Fatalf("cannot create binde for package %s: %v", test.pkgName, err)
		}
		files := bnd.Files()
		if err := compare(files[0], test.wantHeader); err != nil {
			t.Errorf("error in header file generation for package %s: %v", test.pkgName, err)
		}
		if err := compare(files[1], test.wantCC); err != nil {
			t.Errorf("error in header file generation for package %s: %v", test.pkgName, err)
		}
	}
}
