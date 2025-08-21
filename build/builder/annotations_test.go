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

package builder_test

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
)

type idAnnotation struct {
	cpevelements.CoreMacroElement
	fn  ir.PkgFunc
	tag string
}

var _ cpevelements.FuncAnnotator = (*idAnnotation)(nil)

func newAnnotation(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (cpevelements.MacroElement, error) {
	fn, ok := args[0].(ir.PkgFunc)
	if !ok {
		return nil, errors.Errorf("%T not an IR function", args[0])
	}
	var tag string
	switch argT := args[1].(type) {
	case *elements.String:
		tag = argT.StringValue().String()
	case interp.Func:
		tag = argT.Func().Name()
	default:
		tag = fmt.Sprintf("%T", argT)
	}
	return &idAnnotation{
		CoreMacroElement: cpevelements.CoreMacroElement{Mac: macro},
		fn:               fn,
		tag:              tag,
	}, nil
}

func (m *idAnnotation) Annotate(errApp fmterr.ErrAppender, fn ir.PkgFunc) bool {
	fn.Annotations().Append(
		m.Macro().Func().File().Package,
		"TAG",
		m.tag,
	)
	return true
}

func TestAnnotation(t *testing.T) {
	testbuild.Run(t,
		testbuild.DeclarePackage{
			Src: `
package annotation

// gx:irmacro
func Tag(any, any) any
`,
			Post: func(pkg *ir.Package) {
				id := pkg.FindFunc("Tag").(*ir.Macro)
				id.BuildSynthetic = cpevelements.MacroImpl(newAnnotation)
			},
		},
		testbuild.Decl{
			Src: `
import "annotation"

// gx:@annotation.Tag("Hello")
func f() int32 {
	return 2
}
`,
			Want: []ir.Node{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.IntNumberAs(2, ir.Int32Type()),
							},
						},
					),
					Anns: ir.Annotations{
						Anns: []*ir.Annotation{
							ir.NewAnnotation("annotation:TAG", "Hello"),
						},
					},
				},
			},
		},
		testbuild.Decl{
			Src: `
import "annotation"

// gx:@annotation.Tag("Hello")
// gx:@annotation.Tag("Bonjour")
func f() int32 {
	return 2
}
`,
			Want: []ir.Node{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.IntNumberAs(2, ir.Int32Type()),
							},
						},
					),
					Anns: ir.Annotations{
						Anns: []*ir.Annotation{
							ir.NewAnnotation("annotation:TAG", "Hello"),
							ir.NewAnnotation("annotation:TAG", "Bonjour"),
						},
					},
				},
			},
		},
		testbuild.Decl{
			Src: `
import "annotation"

// gx:@annotation.Tag("Hello")
// gx:@annotation.Tag("Bonjour")
// gx:@annotation.Tag(f)
func f() int32
`,
			Want: []ir.Node{
				&ir.FuncBuiltin{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Anns: ir.Annotations{
						Anns: []*ir.Annotation{
							ir.NewAnnotation("annotation:TAG", "Hello"),
							ir.NewAnnotation("annotation:TAG", "Bonjour"),
							ir.NewAnnotation("annotation:TAG", "f"),
						},
					},
				},
			},
		},
	)
}
