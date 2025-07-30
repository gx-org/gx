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

package builder

import (
	"go/ast"
	"strings"
)

type directive int

const (
	invalid directive = iota
	irmacro
	cpeval
	none
)

func (d directive) String() string {
	switch d {
	case none:
		return "none"
	case irmacro:
		return "irmacro"
	case cpeval:
		return "compeval"
	default:
		return "invalid"
	}
}

const directivePrefix = "gx:"

var directives = map[string]directive{
	irmacro.String(): irmacro,
	cpeval.String():  cpeval,
	none.String():    none,
}

func trimCommentPrefix(cmt *ast.Comment) string {
	text := cmt.Text
	text = strings.TrimPrefix(text, "//")
	text = strings.TrimSpace(text)
	return text
}

func processFuncDirective(pscope procScope, fn *ast.FuncDecl) (directive, *ast.Comment, bool) {
	if fn.Doc == nil {
		return none, nil, true
	}
	dir := none
	var comment *ast.Comment
	for _, doc := range fn.Doc.List {
		text := trimCommentPrefix(doc)
		if !strings.HasPrefix(text, directivePrefix) {
			continue
		}
		dirS := text[len(directivePrefix):]
		docDir := directives[dirS]
		if docDir == invalid {
			return dir, doc, pscope.err().Appendf(doc, "undefined directive %s", doc.Text)
		}
		if dir != none {
			return dir, doc, pscope.err().Appendf(doc, "a function can only have one GX directive")
		}
		dir = docDir
		comment = doc
	}
	return dir, comment, true
}
