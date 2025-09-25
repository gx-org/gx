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
	"github.com/gx-org/gx/build/ir/annotations"
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/fun"
)

type tagStr struct {
	s string
}

func newTag() *tagStr {
	return &tagStr{}
}

func (ts *tagStr) append(s string) {
	if ts.s == "" {
		ts.s = s
		return
	}
	ts.s += ", " + s
}

func (ts *tagStr) String() string {
	return ts.s
}

type tagger struct {
	cpevelements.CoreMacroElement
	fn  ir.PkgFunc
	tag string
}

var _ cpevelements.FuncAnnotator = (*tagger)(nil)

func buildTagger(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (cpevelements.MacroElement, error) {
	var tag string
	switch argT := args[0].(type) {
	case *elements.String:
		tag = argT.StringValue().String()
	case fun.Func:
		tag = argT.Func().FuncType().String()
	default:
		tag = fmt.Sprintf("%T", argT)
	}
	return &tagger{
		CoreMacroElement: macro.Element(call),
		tag:              tag,
	}, nil
}

func (m *tagger) Annotate(fetcher ir.Fetcher, fn ir.PkgFunc) bool {
	tg := annotations.GetDef[*tagStr](fn, m.Key(), newTag)
	tg.append(m.tag)
	return true
}

type macroBuildReturn struct {
	cpevelements.CoreMacroElement
	tagMacro *ir.Macro
	tagFn    ir.PkgFunc
}

var _ cpevelements.FuncASTBuilder = (*macroBuildReturn)(nil)

func newBuildReturn(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (cpevelements.MacroElement, error) {
	return &macroBuildReturn{
		CoreMacroElement: macro.Element(call),
		tagMacro:         macro.FindMacro("Tag"),
		tagFn:            args[0].(fun.Func).Func().(ir.PkgFunc),
	}, nil
}

func (m *macroBuildReturn) BuildDecl(origFn ir.PkgFunc) (*ast.FuncDecl, bool) {
	return origFn.Source().(*ast.FuncDecl), true
}

func (m *macroBuildReturn) BuildBody(fetcher ir.Fetcher, fn ir.Func) (*ast.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	ann := annotations.Get[*tagStr](m.tagFn, m.tagMacro)
	return &ast.BlockStmt{List: []ast.Stmt{
		&ast.ReturnStmt{Results: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: ann.s,
			},
		}},
	}}, nil, true
}

func (m *macroBuildReturn) BuildFuncLit(fetcher ir.Fetcher) (*ast.FuncLit, bool) {
	body, _, ok := m.BuildBody(fetcher, nil)
	if !ok {
		return nil, false
	}
	return &ast.FuncLit{
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{
			&ast.Field{
				Type: &ast.Ident{Name: "string"},
			},
		}}},
		Body: body,
	}, true
}

func annotationPackage(tag, buildReturn **ir.Macro) testbuild.DeclarePackage {
	if tag == nil {
		var tagM, buildReturnM *ir.Macro
		tag, buildReturn = &tagM, &buildReturnM
	}
	return testbuild.DeclarePackage{
		Src: `
package annotation

// gx:irmacro
func Tag(any)

// gx:irmacro
func BuildReturn(any) any
`,
		Post: func(pkg *ir.Package) {
			*tag = pkg.FindFunc("Tag").(*ir.Macro)
			(*tag).BuildSynthetic = cpevelements.MacroImpl(buildTagger)
			*buildReturn = pkg.FindFunc("BuildReturn").(*ir.Macro)
			(*buildReturn).BuildSynthetic = cpevelements.MacroImpl(newBuildReturn)
		},
	}
}

func TestAnnotation(t *testing.T) {
	var tag, buildReturn *ir.Macro
	bld := testbuild.Run(t, annotationPackage(&tag, &buildReturn))
	bld.Continue(t,
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
					Anns: irh.Annotations(tag, &tagStr{s: "Hello"}),
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
					Anns: irh.Annotations(tag, &tagStr{s: "Hello, Bonjour"}),
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
					Anns: irh.Annotations(tag, &tagStr{s: "Hello, Bonjour, func() int32"}),
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
					Anns: irh.Annotations(tag, &tagStr{s: "Source"}),
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

func TestAnnotationError(t *testing.T) {
	testbuild.Run(t,
		annotationPackage(nil, nil),
		testbuild.Decl{
			Src: `
import "annotation"

func f() string {
	return annotation.Tag("Source")("Hi")
}
`,
			Err: "invalid use of annotation macro annotation.Tag",
		},
	)
}
