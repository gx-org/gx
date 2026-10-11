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

package fmterr

import (
	"fmt"
	"go/token"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/src/gx/ast"
)

type internal struct {
	err error
}

func (err internal) Error() string {
	return fmt.Sprintf("GX internal error. This is a bug in GX. Please report it. Error:\n%+v", err.err)
}

// IsInternal returns true if the error is an internal GX error.
func IsInternal(err error) bool {
	_, ok := err.(internal)
	return ok
}

// Internal marks an error as internal, potentially adding additional information.
func Internal(err error) error {
	return internal{err: err}
}

// Internalf returns a formatted compiler error for the user.
func Internalf(format string, a ...any) error {
	return Internal(errors.Errorf(format, a...))
}

// InternalAt returns a formatted compiler error for the user at a given code position.
func InternalAt(fset *token.FileSet, src ast.Node, format string, a ...any) error {
	return Internal(Errorf(fset, src, format, a...))
}
