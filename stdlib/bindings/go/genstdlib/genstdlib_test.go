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

package genstdlib_test

import (
	"io"
	"strings"
	"testing"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/bindings/go/genstdlib"
)

type writer struct {
	strings.Builder
}

func (writer) Close() error { return nil }

func TestGenerate(t *testing.T) {
	if err := genstdlib.BuildAll(func(*ir.Package) (io.WriteCloser, error) {
		return &writer{}, nil
	}); err != nil {
		t.Errorf("\n%+v", fmterr.ToStackTraceError(err))
	}

}
