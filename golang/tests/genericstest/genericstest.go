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

// Package genericstest tests the Go bindings of the generics GX package.
package genericstest

import (
	"testing"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/tests/bindings/generics/generics_go_gx"
)

var generics *generics_go_gx.PackageHandle

// TestNewUseS checks that UseS is exported even if its private field uses a generic type.
func TestNewUseS(t *testing.T) {
	generics.Factory.NewUseS()
}

func setupTest(dev *api.Device) error {
	var err error
	generics, err = generics_go_gx.BuildHandleFor(dev)
	return err
}
