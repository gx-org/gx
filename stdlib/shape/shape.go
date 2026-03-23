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

// Package shape provides functions to manipulate the shape of tensors.
package shape

import (
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

// Package description of the GX shape package.
var Package = builtin.PackageBuilder{
	FullPath: "shape",
	Builders: []builtin.Builder{
		builtin.ParseSource(),
		builtin.BuildFunc(concat{}),
		builtin.BuildFunc(split{}),
		builtin.BuildFunc(broadcast{}),
		builtin.BuildFunc(gather{}),
		builtin.ImplementStubFunc("Len", func(impl *impl.Stdlib) interp.FuncBuiltin { return impl.Shapes.Len }),
	},
}
