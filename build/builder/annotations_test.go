package builder_test

import (
	"go/ast"
	"testing"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
)

func newAnnotation(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (*cpevelements.SyntheticFunc, error) {
	fn, ok := args[0].(ir.PkgFunc)
	if !ok {
		return nil, errors.Errorf("%T not an IR function", args[0])
	}
	tag, ok := args[1].(*elements.String)
	if !ok {
		return nil, errors.Errorf("%T is not a string element", args[1])
	}
	return cpevelements.NewSyntheticFunc(&idAnnotation{
		macro: macro,
		fn:    fn,
		tag:   tag.StringValue().String(),
	}), nil
}

type idAnnotation struct {
	macro *cpevelements.Macro
	fn    ir.PkgFunc
	tag   string
}

func (m *idAnnotation) BuildType() (*ast.FuncDecl, error) {
	return &ast.FuncDecl{Type: m.fn.FuncType().Src}, nil
}

func (m *idAnnotation) BuildBody(fetcher ir.Fetcher) (*ast.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	fDecl, ok := m.fn.(*ir.FuncDecl)
	if !ok {
		return nil, nil, true
	}
	return fDecl.Body.Src, nil, true
}

func (m *idAnnotation) BuildIR(errApp fmterr.ErrAppender, src *ast.FuncDecl, file *ir.File, fType *ir.FuncType) (ir.PkgFunc, bool) {
	ext := m.fn.New(src, file, fType)
	ext.Annotations().Append(
		m.macro.Func().File().Package,
		"TAG",
		m.tag,
	)
	return ext, true
}

func TestAnnotation(t *testing.T) {
	testbuild.Run(t,
		testbuild.DeclarePackage{
			Src: `
package annotation

// gx:irmacro
func Tag(any, string) any
`,
			Post: func(pkg *ir.Package) {
				id := pkg.FindFunc("Tag").(*ir.Macro)
				id.BuildSynthetic = cpevelements.MacroImpl(newAnnotation)
			},
		},
		testbuild.Decl{
			Src: `
import "annotation"

// gx@=annotation.Tag("Hello")
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
						Anns: []ir.Annotation{
							ir.NewAnnotation("annotation:TAG", "Hello"),
						},
					},
				},
			},
		},
		testbuild.Decl{
			Src: `
import "annotation"

// gx@=annotation.Tag("Hello")
// gx@=annotation.Tag("Bonjour")
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
						Anns: []ir.Annotation{
							ir.NewAnnotation("annotation:TAG", "Bonjour"),
							ir.NewAnnotation("annotation:TAG", "Hello"),
						},
					},
				},
			},
		},
		testbuild.Decl{
			Src: `
import "annotation"

// gx@=annotation.Tag("Hello")
// gx@=annotation.Tag("Bonjour")
func f() int32
`,
		},
	)
}
