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

// Package cast provides a utility function to cast an object to another.
package cast

import (
	"reflect"

	"github.com/pkg/errors"
)

// To casts an instance to a given type.
func To[T any](x any) (xT T, err error) {
	xT, isT := x.(T)
	if !isT {
		err = errors.Errorf("cannot convert %T to %s", x, reflect.TypeFor[T]().String())
	}
	return
}
