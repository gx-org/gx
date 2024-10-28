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
	"go/ast"
	"go/token"
)

// FileSet builds errors formatted for a given file set.
type FileSet struct {
	FSet *token.FileSet
}

// Errorf returns a formatted compiler error for the user.
func (f FileSet) Errorf(node ast.Node, format string, a ...any) error {
	return Errorf(f.FSet, node, format, a...)
}

// Position positions an error in GX.
func (f FileSet) Position(node ast.Node, err error) error {
	return Position(f.FSet, node, err)
}

// Pos returns a formatter with a fileset and a position as a context.
func (f FileSet) Pos(node ast.Node) Pos {
	return Pos{FileSet: f, Node: node}
}

// Pos builds errors for a position in a file set.
type Pos struct {
	FileSet
	Node ast.Node
}

// Errorf returns a formatted compiler error for the user.
func (f Pos) Errorf(format string, a ...any) error {
	return f.FileSet.Errorf(f.Node, format, a...)
}
