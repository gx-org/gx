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

package fmterr

import (
	"fmt"
	"go/ast"
	"go/token"
	"runtime/debug"

	"github.com/pkg/errors"
)

type (
	// ErrorWithPos is an error attached to a position in GX code.
	ErrorWithPos interface {
		error
		FSet() *token.FileSet
		Pos() Pos
		Err() error
	}

	errorWithPos struct {
		pos Pos
		err error
	}
)

// Error returns an error at a given position.
func Error(fset *token.FileSet, node ast.Node, err error) ErrorWithPos {
	var errPos ErrorWithPos
	if errors.As(err, &errPos) {
		return errPos
	}
	return At(fset, node).Error(err)
}

// Errorf returns a formatted compiler error for the user.
func Errorf(fset *token.FileSet, src ast.Node, format string, a ...any) ErrorWithPos {
	return Error(fset, src, errors.Errorf(format, a...))
}

// Internal marks an error as internal, potentially adding additional information.
func Internal(err error) error {
	return fmt.Errorf("GX internal error. This is a bug in GX. Please report it. Error:\n%+v", err)
}

// Internalf returns a formatted compiler error for the user.
func Internalf(fset *token.FileSet, src ast.Node, format string, a ...any) error {
	err := Errorf(fset, src, format, a...)
	return Internal(err)
}

// Error returns a string description of the error.
func (err errorWithPos) Error() (s string) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		s = fmt.Sprintf("recovered from panic when building error message: %T:\n%v", err.err, string(debug.Stack()))
	}()
	if err.FSet() == nil {
		return err.err.Error()
	}
	return err.pos.String() + " " + err.err.Error()
}

func (err errorWithPos) Pos() Pos {
	return err.pos
}

// Unwrap the error.
func (err errorWithPos) Unwrap() error {
	return err.err
}

// Format writes the error into the state of the formatter.
func (err errorWithPos) Format(s fmt.State, verb rune) {
	format(err, s, verb)
}

func (err errorWithPos) FSet() *token.FileSet {
	return err.pos.FileSet.FSet
}

func (err errorWithPos) Err() error {
	return err.err
}
