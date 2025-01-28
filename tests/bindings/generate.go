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

// Package bindings provide test files to test bindings.
package bindings

//go:generate go run github.com/gx-org/gx/golang/packager --gx_package_module=github.com/gx-org/gx/tests/bindings/basic
//go:generate go run github.com/gx-org/gx/golang/packager --gx_package_module=github.com/gx-org/gx/tests/bindings/encoding
//go:generate go run github.com/gx-org/gx/golang/packager --gx_package_module=github.com/gx-org/gx/tests/bindings/imports
//go:generate go run github.com/gx-org/gx/golang/packager --gx_package_module=github.com/gx-org/gx/tests/bindings/math
//go:generate go run github.com/gx-org/gx/golang/packager --gx_package_module=github.com/gx-org/gx/tests/bindings/parameters
//go:generate go run github.com/gx-org/gx/golang/packager --gx_package_module=github.com/gx-org/gx/tests/bindings/pkgvars
//go:generate go run github.com/gx-org/gx/golang/packager --gx_package_module=github.com/gx-org/gx/tests/bindings/rand
//go:generate go run github.com/gx-org/gx/golang/packager --gx_package_module=github.com/gx-org/gx/tests/bindings/cartpole
//go:generate go run github.com/gx-org/gx/golang/packager --gx_package_module=github.com/gx-org/gx/tests/bindings/dtypes

//go:generate go run github.com/gx-org/gx/golang/binder/genbind --gx_package=github.com/gx-org/gx/tests/bindings/basic
//go:generate go run github.com/gx-org/gx/golang/binder/genbind --gx_package=github.com/gx-org/gx/tests/bindings/encoding
//go:generate go run github.com/gx-org/gx/golang/binder/genbind --gx_package=github.com/gx-org/gx/tests/bindings/imports
//go:generate go run github.com/gx-org/gx/golang/binder/genbind --gx_package=github.com/gx-org/gx/tests/bindings/math
//go:generate go run github.com/gx-org/gx/golang/binder/genbind --gx_package=github.com/gx-org/gx/tests/bindings/parameters
//go:generate go run github.com/gx-org/gx/golang/binder/genbind --gx_package=github.com/gx-org/gx/tests/bindings/pkgvars
//go:generate go run github.com/gx-org/gx/golang/binder/genbind --gx_package=github.com/gx-org/gx/tests/bindings/rand
//go:generate go run github.com/gx-org/gx/golang/binder/genbind --gx_package=github.com/gx-org/gx/tests/bindings/cartpole
//go:generate go run github.com/gx-org/gx/golang/binder/genbind --gx_package=github.com/gx-org/gx/tests/bindings/dtypes
