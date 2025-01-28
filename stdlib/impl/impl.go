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

// Package impl provides a structure pointing to standard library functions provided by a backend.
package impl

import "github.com/gx-org/gx/interp"

type (
	// Stdlib is an implementation of the standard library functions by a backend.
	Stdlib struct {
		Control Control
		Math    Math
		Num     Num
		Rand    Rand
		Shapes  Shapes
	}

	// Control is the implementation of the control package.
	Control struct {
		While interp.FuncBuiltin
	}

	// Math is the implementation of the math package.
	Math struct {
		Pow  interp.FuncBuiltin
		Exp  interp.FuncBuiltin
		Log  interp.FuncBuiltin
		Min  interp.FuncBuiltin
		Max  interp.FuncBuiltin
		Cos  interp.FuncBuiltin
		Sin  interp.FuncBuiltin
		Sqrt interp.FuncBuiltin
		Tanh interp.FuncBuiltin
		Ceil interp.FuncBuiltin
	}

	// Num is the implementation of the num package.
	Num struct {
		Iota      interp.FuncBuiltin
		IotaFull  interp.FuncBuiltin
		Einsum    interp.FuncBuiltin
		MatMul    interp.FuncBuiltin
		Sum       interp.FuncBuiltin
		Transpose interp.FuncBuiltin
		ReduceMax interp.FuncBuiltin
		Argmax    interp.FuncBuiltin
	}

	// Rand of the rand package
	Rand struct {
		BootstrapGeneratorNew  interp.FuncBuiltin
		BootstrapGeneratorNext interp.FuncBuiltin
		PhiloxUint32           interp.FuncBuiltin
		PhiloxUint64           interp.FuncBuiltin
	}

	// Shapes is the implementation of the shapes package.
	Shapes struct {
		Concat interp.FuncBuiltin
		Len    interp.FuncBuiltin
		Split  interp.FuncBuiltin
		Expand interp.FuncBuiltin
		Gather interp.FuncBuiltin
	}
)
