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
	"go/ast"
	"go/token"
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
)

type (
	tagAnnotation ir.AnnotationT[string]

	tagger struct {
		cpevelements.CoreMacroElement
		fn  ir.PkgFunc
		tag string
	}
)

var _ cpevelements.FuncAnnotator = (*tagger)(nil)

const tagKey = "annotation:TAG"

func buildTagger(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (cpevelements.MacroElement, error) {
	var tag string
	switch argT := args[0].(type) {
	case *elements.String:
		tag = argT.StringValue().String()
	case interp.Func:
		tag = argT.Func().Name()
	default:
		tag = fmt.Sprintf("%T", argT)
	}
	return &tagger{
		CoreMacroElement: cpevelements.CoreMacroElement{Mac: macro},
		tag:              tag,
	}, nil
}

func (m *tagger) Annotate(fetcher ir.Fetcher, fn ir.PkgFunc) bool {
	fn.Annotations().Set(ir.NewAnnotation(tagKey, m.tag))
	return true
}

type macroBuildReturn struct {
	cpevelements.CoreMacroElement
	tagFn ir.PkgFunc
}

var _ cpevelements.FuncASTBuilder = (*macroBuildReturn)(nil)

func newBuildReturn(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (cpevelements.MacroElement, error) {
	return &macroBuildReturn{
		CoreMacroElement: cpevelements.CoreMacroElement{Mac: macro},
		tagFn:            args[0].(interp.Func).Func().(ir.PkgFunc),
	}, nil
}

func (m *macroBuildReturn) BuildDecl(origFn ir.PkgFunc) (*ast.FuncDecl, bool) {
	return origFn.Source().(*ast.FuncDecl), true
}

func (m *macroBuildReturn) BuildBody(fetcher ir.Fetcher, fn ir.Func) (*ast.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	return &ast.BlockStmt{List: []ast.Stmt{
		&ast.ReturnStmt{Results: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: ir.AnnotationFrom[string](m.tagFn, tagKey).Value(),
			},
		}},
	}}, nil, true
}

func TestAnnotation(t *testing.T) {
	testbuild.Run(t,
		testbuild.DeclarePackage{
			Src: `
package annotation

// gx:irmacro
func Tag(any) any

// gx:irmacro
func BuildReturn(any) any
`,
			Post: func(pkg *ir.Package) {
				id := pkg.FindFunc("Tag").(*ir.Macro)
				id.BuildSynthetic = cpevelements.MacroImpl(buildTagger)
				id = pkg.FindFunc("BuildReturn").(*ir.Macro)
				id.BuildSynthetic = cpevelements.MacroImpl(newBuildReturn)
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
					Anns: irh.Annotations(
						ir.NewAnnotation("annotation:TAG", "Hello"),
					),
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
					Anns: irh.Annotations(
						ir.NewAnnotation("annotation:TAG", "Hello"),
						ir.NewAnnotation("annotation:TAG", "Bonjour"),
					),
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
					Anns: irh.Annotations(
						ir.NewAnnotation("annotation:TAG", "Hello"),
						ir.NewAnnotation("annotation:TAG", "Bonjour"),
						ir.NewAnnotation("annotation:TAG", "f"),
					),
				},
			},
		},
		testbuild.Decl{
			Src: `
import "annotation"

// gx:=annotation.BuildReturn(f)
func g() string

// gx:@annotation.Tag("Source")
func f() int32
`,
			Want: []ir.Node{
				&ir.FuncBuiltin{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Anns: irh.Annotations(
						ir.NewAnnotation("annotation:TAG", "Source"),
					),
				},
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.StringType()),
					),
					Body: irh.SingleReturn(&ir.StringLiteral{
						Src: &ast.BasicLit{Value: "Source"},
					}),
				},
			},
		},
	)
}
