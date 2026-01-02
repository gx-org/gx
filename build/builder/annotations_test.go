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
	"strconv"
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

var tagKey = annotations.NewKey(tagStr{})

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

func tagFunc(fetcher ir.Fetcher, annotator *ir.Annotator, fn ir.PkgFunc, call *ir.FuncCallExpr, args []ir.Element) bool {
	var tag string
	switch argT := args[0].(type) {
	case *elements.String:
		tag = argT.StringValue().String()
	case fun.Func:
		tag = argT.Func().FuncType().String()
	default:
		tag = fmt.Sprintf("%T", argT)
	}
	tg := annotations.GetDef[*tagStr](fn, tagKey, newTag)
	tg.append(tag)
	return true
}

type macroBuildReturn struct {
	cpevelements.CoreMacroElement
	tagFn ir.PkgFunc
}

var _ ir.FuncASTBuilder = (*macroBuildReturn)(nil)

func newBuildReturn(file *ir.File, call *ir.FuncCallExpr, macro *ir.Macro, args []ir.Element) (ir.MacroElement, error) {
	return &macroBuildReturn{
		CoreMacroElement: cpevelements.MacroElement(macro, file, call),
		tagFn:            args[0].(fun.Func).Func().(ir.PkgFunc),
	}, nil
}

func (m *macroBuildReturn) BuildDecl(origFn ir.PkgFunc) (*ir.File, *ast.FuncDecl, bool) {
	return m.tagFn.File(), origFn.Node().(*ast.FuncDecl), true
}

func (m *macroBuildReturn) BuildBody(fetcher ir.Fetcher, fn ir.Func) (*ast.BlockStmt, bool) {
	ann := annotations.Get[*tagStr](m.tagFn, tagKey)
	if ann == nil {
		return nil, fetcher.Err().Appendf(m.Source(), "cannot find tag value on function %s", m.tagFn.ShortString())
	}
	return &ast.BlockStmt{List: []ast.Stmt{
		&ast.ReturnStmt{Results: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(ann.s),
			},
		}},
	}}, true
}

func annotationPackage(tag **ir.Annotator, buildReturn **ir.Macro) testbuild.DeclarePackage {
	if tag == nil {
		var tagM *ir.Annotator
		tag = &tagM
		var buildReturnM *ir.Macro
		buildReturn = &buildReturnM
	}
	return testbuild.DeclarePackage{
		Src: `
package annotation

// gx:annotator
func Tag(any)

// gx:irmacro
func BuildReturn(any) any
`,
		Post: func(pkg *ir.Package) {
			*tag = pkg.FindFunc("Tag").(*ir.Annotator)
			(*tag).Annotate = ir.AnnotatorImpl(tagFunc)
			*buildReturn = pkg.FindFunc("BuildReturn").(*ir.Macro)
			(*buildReturn).BuildSynthetic = ir.MacroImpl(newBuildReturn)
		},
	}
}

func TestAnnotation(t *testing.T) {
	var tag *ir.Annotator
	var buildReturn *ir.Macro
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
			Want: []ir.IR{
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
					Anns: irh.Annotations(tagKey, &tagStr{s: "Hello"}),
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
			Want: []ir.IR{
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
					Anns: irh.Annotations(tagKey, &tagStr{s: "Hello, Bonjour"}),
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
			Want: []ir.IR{
				&ir.FuncBuiltin{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Anns: irh.Annotations(tagKey, &tagStr{s: "Hello, Bonjour, func() int32"}),
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
			Want: []ir.IR{
				&ir.FuncBuiltin{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Anns: irh.Annotations(tagKey, &tagStr{s: "Source"}),
				},
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.StringType()),
					),
					Body: irh.SingleReturn(&ir.StringLiteral{
						Src: &ast.BasicLit{Value: strconv.Quote("Source")},
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
			Err: "annotator gx:@annotation.Tag only valid in a function annotation context",
		},
	)
}
