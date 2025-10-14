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

// Package fmterr provides helpers to accumulate errors while compiling and
// format errors given a position a fileset.
package fmterr

import (
	"fmt"
	"go/token"
)

// PrefixWith returns a function to prefix errors with a formatted string.
func PrefixWith(s string, o ...any) func(err error) error {
	return func(err error) error {
		return fmt.Errorf("%s%w", fmt.Sprintf(s, o...), err)
	}
}

// PosPrefixWith returns a function to prefix errors with a formatted string.
func PosPrefixWith(fset *token.FileSet, pos token.Pos, s string, o ...any) func(err error) error {
	return func(err error) error {
		return fmt.Errorf("%s%s%w", PosString(fset, pos), fmt.Sprintf(s, o...), err)
	}
}
