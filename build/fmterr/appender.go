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
	"go/ast"
)

// Appender appends errors to a set within the context of a FileSet.
type Appender struct {
	errors *Errors
	fset   FileSet
}

// Append an error to the list of errors.
func (app *Appender) Append(err error) bool {
	app.errors.Append(err)
	return false
}

// AppendAt appends an existing error at a given position.
func (app *Appender) AppendAt(node ast.Node, err error) bool {
	app.Append(app.fset.Position(node, err))
	return false
}

// Appendf appends an error at a position.
func (app *Appender) Appendf(node ast.Node, format string, a ...any) bool {
	app.Append(app.fset.Errorf(node, format, a...))
	return false
}

// AppendInternalf appends an internal error at a position.
func (app *Appender) AppendInternalf(node ast.Node, format string, a ...any) bool {
	app.Append(Internal(app.fset.Errorf(node, format, a...), ""))
	return false
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
	if app.errors.Empty() {
		return nil
	}
	return app.errors
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
	app.Append(Internal(app.pos.Errorf(format, a...), ""))
}
