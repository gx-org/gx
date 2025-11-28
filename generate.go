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

// Package gx is only used to generate C source code.
package gx

//go:generate mkdir -p gxdeps/github.com/gx-org
//go:generate bash -c "[ -d 'gxdeps/github.com/gx-org/gx' ] || ln -sf ../../.. gxdeps/github.com/gx-org/gx"
//go:generate go tool cgo -exportheader golang/binder/cgx/cgx.cgo.h golang/binder/cgx/cgx.go
//go:generate go tool cgo -exportheader golang/binder/cgx/testing/testing.cgo.h golang/binder/cgx/testing/testing.go
