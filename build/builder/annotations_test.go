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

func toTagString(arg ir.Element) string {
	switch argT := arg.(type) {
	case *elements.String:
		return argT.StringValue().String()
	case fun.Func:
		return argT.Func().FuncType().String()
	default:
		return fmt.Sprintf("%T", argT)
	}
}

func tagFunc(fetcher ir.Fetcher, annotator *ir.AnnotatorFunc, fn ir.PkgFunc, call *ir.FuncCallExpr, args []ir.Element) bool {
	tg := annotations.GetDef[*tagStr](fn, tagKey, newTag)
	tg.append(toTagString(args[0]))
	return true
}

type tagCheck struct{}

var tagChecker = &tagCheck{}

func (tagCheck) Check(fetcher ir.Fetcher, list *ir.FieldList) bool {
	found := make(map[string]*ir.FieldGroup)
	ok := true
	for _, grp := range list.List {
		tag := annotations.Get[*tagStr](grp, tagKey)
		if tag == nil {
			continue
		}
		if prev := found[tag.s]; prev != nil {
			ok = fetcher.Err().Appendf(grp.Src, "tag %s has already been used", tag.s)
			continue
		}
		found[tag.s] = grp
	}
	return ok
}

func tagField(fetcher ir.Fetcher, annotator *ir.AnnotatorField, field *ir.FieldGroup, call *ir.FuncCallExpr, args []ir.Element) (ir.FieldListCheckImpl, bool) {
	tg := annotations.GetDef[*tagStr](field, tagKey, newTag)
	tg.append(toTagString(args[0]))
	return tagChecker, true
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

var annotationPackage = testbuild.DeclarePackage{
	Src: `
package annotation

// gx:funcAnnotator
func Tag(any)

// gx:fieldAnnotator
func TagField(any)

// gx:irmacro
func BuildReturn(any) any
`,
	Post: func(pkg *ir.Package) {
		tag := pkg.FindFunc("Tag").(*ir.AnnotatorFunc)
		tag.Annotate = ir.AnnotatorFuncImpl(tagFunc)
		tagFld := pkg.FindFunc("TagField").(*ir.AnnotatorField)
		tagFld.Annotate = ir.AnnotatorFieldImpl(tagField)
		buildReturn := pkg.FindFunc("BuildReturn").(*ir.Macro)
		buildReturn.BuildSynthetic = ir.MacroImpl(newBuildReturn)
	},
}

func TestFuncAnnotation(t *testing.T) {
	testbuild.Run(t,
		annotationPackage,
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

func TestFuncAnnotationError(t *testing.T) {
	testbuild.Run(t,
		annotationPackage,
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

func TestFieldAnnotation(t *testing.T) {
	testbuild.Run(t,
		annotationPackage,
		testbuild.Decl{
			Src: `
import "annotation"

type S struct {	
	A int32 ` + "`gx:@annotation.TagField(\"hello\")`" + `
	B int32 ` + "`gx:@annotation.TagField(\"bonjour\")`" + `
}
`,
			Want: []ir.IR{
				&ir.NamedType{
					File: wantFile,
					Src:  &ast.TypeSpec{Name: irh.IdentAST("A")},
					Underlying: ir.TypeExpr(nil, irh.StructType(
						irh.Field(
							"A", nil,
							&ir.FieldGroup{
								Type: ir.TypeExpr(nil, ir.Int32Type()),
								Anns: irh.Annotations(tagKey, &tagStr{s: "hello"}),
							}),
						irh.Field(
							"B", nil,
							&ir.FieldGroup{
								Type: ir.TypeExpr(nil, ir.Int32Type()),
								Anns: irh.Annotations(tagKey, &tagStr{s: "bonjour"}),
							}),
					)),
				},
			},
		},
	)
}

func TestFieldAnnotationError(t *testing.T) {
	testbuild.Run(t,
		annotationPackage,
		testbuild.Decl{
			Src: `
import "annotation"

type S struct {	
	A int32 ` + "`gx:annotation.Tag(\"params\")`" + `
}
`,
			Err: "expect annotation gx:@... but found gx:annotation.Tag",
		},
		testbuild.Decl{
			Src: `
import "annotation"

type S struct {	
	A int32 ` + "`gx:@annotation.Tag(\"params\")`" + `
}
`,
			Err: "cannot use annotation.Tag to annotate the field of a structure",
		},
		testbuild.Decl{
			Src: `
import "annotation"

type S struct {	
	A int32 ` + "`gx:@annotation.TagField(\"hello\")`" + `
	B int32 ` + "`gx:@annotation.TagField(\"hello\")`" + `
}
`,
			Err: "tag hello has already been used",
		},
	)
}
