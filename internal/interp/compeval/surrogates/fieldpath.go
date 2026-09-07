// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package surrogates

import (
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/interp/elements"
)

type fieldPath struct {
	core
}

var _ elements.FieldPath = &fieldPath{}

func newFieldPath(path storepath.Path) Element {
	return &fieldPath{core: core{path: path}}
}

// Type returns an invalid type.
func (*fieldPath) Type() ir.Type {
	return ir.FieldPathType()
}

func (*fieldPath) FollowOn(ir.Element) (ir.Element, error) {
	return elements.Unknown(), nil
}

func (f *fieldPath) Root() (*storepath.Proxy, error) {
	return nil, fmterr.Internalf("unexpected call to surrogates.fieldPath.Root")
}

func (f *fieldPath) Store() ir.Storage {
	return f.path.Store()
}
