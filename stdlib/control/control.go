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

// Package control provides runtime control flow via the GX `control` standard library.
package control

import (
	"github.com/gx-org/gx/stdlib/builtin"
)

// Package description of the GX control package.
var Package = builtin.PackageBuilder{
	FullPath: "control",
	Builders: []builtin.Builder{
		builtin.BuildFunc(while{}),
	},
}
