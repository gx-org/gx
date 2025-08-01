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

// Package num provides the functions in the num GX standard library.
package num

import (
	"embed"

	"github.com/gx-org/gx/stdlib/builtin"
)

//go:embed *.gx
var fs embed.FS

// Package description of the GX num package.
var Package = builtin.PackageBuilder{
	FullPath: "num",
	Builders: []builtin.Builder{
		builtin.ParseSource(&fs, "num.gx"),
		builtin.BuildFunc(reduceSum{}),
		builtin.BuildFunc(transpose{}),
		builtin.BuildFunc(einsum{}),
		builtin.BuildFunc(matmul{}),
		builtin.BuildFunc(iotaWithAxis{}),
		builtin.BuildFunc(reduceMax{}),
		builtin.BuildFunc(argmax{}),
		builtin.ImplementGraphFunc("IotaFull", evalIotaFull),
	},
}
