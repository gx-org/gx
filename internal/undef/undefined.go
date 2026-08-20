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

// Package undef provides a singleton error to return when the evaluator cannot evaluate an expression because some elements are undefined.
package undef

import "github.com/pkg/errors"

var undefined = errors.Errorf("undefined evaluation")

// Err returns an undefined evaluation error.
func Err() error {
	return undefined
}

// Is returns true if the error is an undefined evaluation.
func Is(err error) bool {
	return undefined == err
}
