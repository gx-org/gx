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

	"github.com/pkg/errors"
)

// FileSet builds errors formatted for a given file set.
type FileSet struct {
	FSet *token.FileSet
}

// Pos returns a formatter with a fileset and a position as a context.
func (f FileSet) Pos(node ast.Node) Pos {
	return At(f.FSet, node)
}

// Pos builds errors for a position in a file set.
type Pos struct {
	Begin token.Position
	End   token.Position
}

// At returns a position given the file set and node.
func At(fset *token.FileSet, node ast.Node) Pos {
	return Pos{
		Begin: fset.Position(node.Pos()),
		End:   fset.Position(node.Pos()),
	}
}

// Error returns a new error at the position.
func (f Pos) Error(err error) ErrorWithPos {
	return errorWithPos{
		pos: f,
		err: err,
	}
}

// Position positions an error in GX.
func (f Pos) Position() token.Position {
	return f.Begin
}

// Errorf returns a formatted compiler error for the user.
func (f Pos) Errorf(format string, a ...any) error {
	return f.Error(errors.Errorf(format, a...))
}

// PosString returns a position as a string that can be used for an error.
func (f Pos) String() string {
	pos := f.Position()
	return fmt.Sprintf("%s:%d:%d", pos.Filename, pos.Line, pos.Column)
}
