package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
)

func newIDMacro(call elements.CallAt, macro *cpevelements.Macro, args []elements.Element) (*cpevelements.SyntheticFunc, error) {
	fn, err := cpevelements.FuncDeclFromElement(args[0])
	if err != nil {
		return nil, err
	}
	return cpevelements.NewSyntheticFunc(macro, &idMacro{fn: fn}), nil
}

type idMacro struct {
	fn *ir.FuncDecl
}

func (m *idMacro) BuildType() (*ir.FuncType, error) {
	return m.fn.FType, nil
}

func (m *idMacro) BuildBody() (*ir.BlockStmt, error) {
	return m.fn.Body, nil
}

func TestMacro(t *testing.T) {
	testAll(t,
		declarePackage{
			src: `
package macro 

//gx:irmacro
func ID(any) any
`,
			post: func(pkg *ir.Package) {
				id := pkg.FindFunc("ID").(*ir.Macro)
				id.BuildSynthetic = cpevelements.MacroImpl(newIDMacro)
			},
		},
		irDeclTest{
			src: `
import "macro"

//gx:=macro.ID(f)
func synthetic()

func f() int32 {
	return 2
}
`,
			want: []ir.Node{
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
					)},
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
					)},
			},
		},
	)
}
