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
	"io"

	"github.com/pkg/errors"
)

type (
	// ErrorWithPos is an error attached to a position in GX code.
	ErrorWithPos interface {
		error
		FSet() *token.FileSet
		Src() ast.Node
		Err() error
	}

	errorWithPos struct {
		fset *token.FileSet
		src  ast.Node
		err  error
	}
)

// Position adds GX position information to an error.
func Position(fset *token.FileSet, src ast.Node, err error) ErrorWithPos {
	return errorWithPos{
		fset: fset,
		src:  src,
		err:  err,
	}
}

// Errorf returns a formatted compiler error for the user.
func Errorf(fset *token.FileSet, src ast.Node, format string, a ...any) error {
	return Position(fset, src, errors.Errorf(format, a...))
}

type internalError struct {
	msg string
	err error
}

func (e *internalError) prefix() string {
	const internalPrefix = "GX internal error. This is a bug in GX. Please report it. Error:\n"
	if e.msg == "" {
		return internalPrefix
	}
	return fmt.Sprintf("%s%s\nOriginal error:\n", internalPrefix, e.msg)
}

func (e *internalError) Error() string {
	return e.prefix() + e.err.Error()
}

func (e *internalError) Unwrap() error {
	return e.err
}

func (e *internalError) Format(s fmt.State, verb rune) {
	io.WriteString(s, e.prefix())
	format(e.err, s, verb)
}

// Internal marks an error as internal, potentially adding additional information.
func Internal(err error, format string, a ...any) error {
	return &internalError{
		msg: fmt.Sprintf(format, a...),
		err: err,
	}
}

// Internalf returns a formatted compiler error for the user.
func Internalf(fset *token.FileSet, src ast.Node, format string, a ...any) error {
	err := Errorf(fset, src, format, a...)
	return Internal(err, "")
}

// Error returns a string description of the error.
func (err errorWithPos) Error() string {
	if err.src == nil {
		return err.err.Error()
	}
	return PosString(err.fset, err.src.Pos()) + " " + err.err.Error()
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
	return err.fset
}

func (err errorWithPos) Src() ast.Node {
	return err.src
}

func (err errorWithPos) Err() error {
	return err.err
}

// PosString returns a position as a string that can be used for an error.
func PosString(fset *token.FileSet, pos token.Pos) string {
	return fset.Position(pos).String() + ":"
}
