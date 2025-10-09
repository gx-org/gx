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
	"errors"
	"go/ast"
)

type (
	// ErrAppender accumulates errors.
	ErrAppender interface {
		// Err returns the accumulator.
		Err() *Appender
	}

	contextError struct {
		f      func(error) error
		errors Errors
	}

	// Appender appends errors to a set within the context of a FileSet.
	Appender struct {
		stack  []contextError
		errors *Errors
		fset   FileSet
	}
)

// Push a new context in the error stack.
func (app *Appender) Push(f func(error) error) {
	app.stack = append(app.stack, contextError{f: f})
}

// Pop removes the last error context in the stack.
func (app *Appender) Pop() {
	last := app.stack[len(app.stack)-1]
	app.stack = app.stack[:len(app.stack)-1]
	if last.errors.Empty() {
		return
	}
	app.Append(last.f(&last.errors))
}

// Append an error to the list of errors.
func (app *Appender) Append(err error) bool {
	if len(app.stack) == 0 {
		app.errors.Append(err)
	} else {
		app.stack[len(app.stack)-1].errors.Append(err)
	}
	return false
}

// AppendAt appends an existing error at a given position.
func (app *Appender) AppendAt(node ast.Node, err error) bool {
	return app.Append(app.fset.Position(node, err))
}

// Appendf appends an error at a position.
func (app *Appender) Appendf(node ast.Node, format string, a ...any) bool {
	return app.Append(app.fset.Errorf(node, format, a...))
}

// AppendInternalf appends an internal error at a position.
func (app *Appender) AppendInternalf(node ast.Node, format string, a ...any) bool {
	return app.Append(Internal(app.fset.Errorf(node, format, a...)))
}

// FSet returns the error fileset formatter.
func (app *Appender) FSet() FileSet {
	return app.fset
}

// Pos returns an appender to a specific position.
func (app *Appender) Pos(node ast.Node) *PosAppender {
	return &PosAppender{app: app, pos: app.fset.Pos(node)}
}

// Errors returns the set of errors or nil if no errors has been appended.
func (app *Appender) Errors() *Errors {
	if len(app.stack) > 0 {
		var errs Errors
		errs.Append(Internal(errors.New("cannot fetch errors while the context stack is non-empty")))
		return &errs
	}
	if app.errors.Empty() {
		return nil
	}
	return app.errors
}

// Empty returns true if no errors has been appended.
func (app *Appender) Empty() bool {
	empty := app.errors.Empty()
	if !empty {
		return false
	}
	for _, app := range app.stack {
		if !app.errors.Empty() {
			return false
		}
	}
	return true
}

// String representation of the error.
func (app *Appender) String() string {
	return app.errors.String()
}

// PosAppender is an error appender for a given position.
type PosAppender struct {
	app *Appender
	pos Pos
}

// Append appends an error at a position.
func (app *PosAppender) Append(err error) {
	app.app.Append(err)
}

// Appendf appends an error at a position.
func (app *PosAppender) Appendf(format string, a ...any) {
	app.app.Append(app.pos.Errorf(format, a...))
}

// AppendInternalf appends an internal error at a position.
func (app *PosAppender) AppendInternalf(format string, a ...any) {
	app.Append(Internal(app.pos.Errorf(format, a...)))
}
