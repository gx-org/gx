// Copyright 2025 Google LLC
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

package ir

import (
	"fmt"
	"strings"
)

type indentBuilder struct {
	b strings.Builder
}

func (g *FieldGroup) String() string {
	var fNames []string
	for _, field := range g.Fields {
		fNames = append(fNames, field.Name.Name)
	}
	typ := g.Type.String()
	names := ""
	if len(fNames) > 0 {
		names = strings.Join(fNames, ", ") + " "
	}
	return names + typ
}

func (l *FieldList) String() string {
	fields := make([]string, len(l.List))
	for i, field := range l.List {
		fields[i] = field.String()
	}
	return strings.Join(fields, ", ")
}

func (b *BlockStmt) String() string {
	stmts := make([]string, len(b.List))
	for i, stmt := range b.List {
		stmts[i] = stmt.String()
	}
	return strings.Join(stmts, "\n") + "\n"
}

func indent(s string) string {
	var lines []string
	for line := range strings.Lines(s) {
		lines = append(lines, "\t"+line)
	}
	return strings.Join(lines, "\n")
}

// String returns a string representation of the function.
func (f *FuncDecl) String() string {
	params := f.FType.Params.String()
	results := f.FType.Results.String()
	body := indent(f.Body.String())
	return fmt.Sprintf("func %s(%s) %s {\n%s}", f.Src.Name.Name, params, results, body)
}

func (s *ReturnStmt) String() string {
	exprs := make([]string, len(s.Results))
	for i, expr := range s.Results {
		exprs[i] = expr.String()
	}
	return fmt.Sprintf("return %s", strings.Join(exprs, ", "))
}

func (*AssignCallStmt) String() string {
	return "AssignCallStmt"
}

func (*AssignExprStmt) String() string {
	return "AssignExprStmt"
}

func (*RangeStmt) String() string {
	return "RangeStmt"
}

func (*IfStmt) String() string {
	return "IfStmt"
}

func (e *ExprStmt) String() string {
	return e.X.String()
}

func (s *ArrayLitExpr) String() string {
	vals := make([]string, len(s.Vals))
	for i, val := range s.Vals {
		vals[i] = val.String()
	}
	return fmt.Sprintf("%s{%s}", s.Type().String(), strings.Join(vals, ", "))
}
