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

// Package embed exposes a builder loading GX files from the files embedded in the binary.
package embed

import (
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers/embedpkg"
	"github.com/gx-org/gx/cgx/handle"
)

// #cgo CFLAGS: -I ../../../..
// #include <gxdeps/github.com/gx-org/gx/golang/binder/cgx/cgx.h>
import "C"

//export cgx_new_embed_builder
func cgx_new_embed_builder() C.cgx_builder {
	return C.cgx_builder(handle.Wrap[*builder.Builder](embedpkg.NewBuilder(nil)))
}
