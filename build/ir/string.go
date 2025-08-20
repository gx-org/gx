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
	"go/token"
	"slices"
	"strings"

	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/base/stringseq"
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
	return stringseq.JoinStringer(slices.Values(l.List), ", ")
}

func (b *BlockStmt) String() string {
	return stringseq.JoinStringer(slices.Values(b.List), "\n") + "\n"
}

// String returns a string representation of the builtin function.
func (f *FuncBuiltin) String() string {
	return f.FType.NameString(f.Src.Name.Name)
}

// String returns a string representation of the function.
func (f *FuncDecl) String() string {
	sig := f.FType.NameString(f.Src.Name.Name)
	body := gxfmt.Indent(f.Body.String())
	return fmt.Sprintf("%s {\n%s}", sig, body)
}

// NameString returns a string representation of a signature given a name.
// The name can be empty.
func (s *FuncType) NameString(name string) string {
	var b strings.Builder
	b.WriteString("func")
	if s.Receiver != nil {
		fmt.Fprintf(&b, " (%s)", s.Receiver.String())
	}
	if name != "" {
		b.WriteString(" " + name)
	}
	if s.TypeParams != nil {
		fmt.Fprintf(&b, "[%s]", s.TypeParams.String())
	}
	b.WriteString("(" + s.Params.String() + ")")
	b.WriteRune(' ')
	if s.Results.Len() > 1 {
		b.WriteString("(")
	}
	b.WriteString(s.Results.String())
	if s.Results.Len() > 1 {
		b.WriteString(")")
	}
	return b.String()
}

// String representation of the type.
func (s *FuncType) String() string {
	return s.NameString("")
}

func (s *ReturnStmt) String() string {
	return "return " + stringseq.JoinStringer(slices.Values(s.Results), ", ")
}

func (s *AssignCallStmt) String() string {
	assigned := make([]string, len(s.List))
	for i, v := range s.List {
		assigned[i] = v.NameDef().Name
	}
	return fmt.Sprintf("%s := %s", strings.Join(assigned, ", "), s.Call.String())
}

func (s *AssignExprStmt) String() string {
	assigned := make([]string, len(s.List))
	exprs := make([]string, len(s.List))
	for i, v := range s.List {
		assigned[i] = v.NameDef().Name
		exprs[i] = v.X.String()
	}
	return fmt.Sprintf("%s %s %s", strings.Join(assigned, ", "), s.Src.Tok, strings.Join(exprs, ", "))
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

// String representation.
func (s *CallExpr) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s(", s.Callee.String())
	stringseq.AppendStringer(&b, slices.Values(s.Args), ", ")
	fmt.Fprintf(&b, ")")
	return b.String()
}

func stringLiteral(elts []AssignableExpr) string {
	ss := make([]string, len(elts))
	for i, elt := range elts {
		ss[i] = elt.String()
	}
	return fmt.Sprintf("{%s}", strings.Join(ss, ", "))
}

func (s *ArrayLitExpr) String() string {
	return fmt.Sprintf("%s%s", s.Typ.String(), stringLiteral(s.Elts))
}

// String representation.
func (s *SliceLitExpr) String() string {
	return fmt.Sprintf("%s%s", s.Typ.String(), stringLiteral(s.Elts))
}

func (e *CastExpr) String() string {
	switch e.Typ.(type) {
	case *NamedType:
		return fmt.Sprintf("%s(%s)", e.Typ.String(), e.X.String())
	case ArrayType:
		return fmt.Sprintf("%s(%s)", e.Typ.String(), e.X.String())
	default:
		return fmt.Sprintf("(%s)(%s)", e.Typ.String(), e.X.String())
	}
}

func (e *TypeAssertExpr) String() string {
	return fmt.Sprintf("%s.(%s)", e.X.String(), e.Typ.String())
}

func (e *NumberFloat) String() string {
	if e.Src != nil {
		return e.Src.Value
	}
	return e.Val.String()
}

func (e *NumberInt) String() string {
	if e.Src != nil && e.Src.Value != "" {
		return e.Src.Value
	}
	return e.Val.String()
}

// String representation of the tuple.
func (e *Tuple) String() string {
	exprs := make([]string, len(e.Exprs))
	for i, expr := range e.Exprs {
		exprs[i] = expr.String()
	}
	return fmt.Sprintf("(%s)", strings.Join(exprs, ","))
}

// Type returns the type of an expression.
func (e *ConstExpr) String() string {
	return fmt.Sprintf("const %s", e.VName.Name)
}

func (dm *AxisInfer) axExprString() string {
	return dm.X.axExprString()
}

// String representation of the dimension.
func (dm *AxisInfer) String() string {
	if dm.X == nil {
		return "[_]"
	}
	return fmt.Sprintf("[%s]", dm.axExprString())
}

func (dm *AxisExpr) axExprString() string {
	return dm.X.String()
}

// String representation of the dimension.
func (dm *AxisExpr) String() string {
	return fmt.Sprintf("[%s]", dm.axExprString())
}

func (dm *AxLengthName) axExprString() string {
	return dm.Src.Name
}

// String representation of the dimension.
func (dm *AxLengthName) String() string {
	return fmt.Sprintf("[%s]", dm.axExprString())
}

// String returns a string representation of the rank.
func (r *Rank) String() string {
	if r == nil {
		return ""
	}
	bld := strings.Builder{}
	for _, dim := range r.Ax {
		bld.WriteString(dim.String())
	}
	return bld.String()
}

// String returns a string representation of the rank.
func (r *RankInfer) String() string {
	if r.Rnk == nil {
		return "[...]"
	}
	return r.Rnk.String()
}

// String representation.
func (s *SelectorExpr) String() string {
	return fmt.Sprintf("%s.%s", s.X.String(), s.Src.Sel.Name)
}

// String representation.
func (s *IndexExpr) String() string {
	return fmt.Sprintf("%s[%s]", s.X.String(), s.Index.String())
}

func (s *BinaryExpr) binaryElementString(x Expr) string {
	bx, isXBinary := x.(*BinaryExpr)
	if !isXBinary {
		return x.String()
	}
	if (s.Src.Op == token.SUB || s.Src.Op == token.ADD) &&
		(bx.Src.Op == token.MUL || bx.Src.Op == token.QUO) {
		return x.String()
	}
	return fmt.Sprintf("(%s)", x.String())
}

// String representation.
func (s *BinaryExpr) String() string {
	return (s.binaryElementString(s.X) +
		s.Src.Op.String() +
		s.binaryElementString(s.Y))
}

// TypeString returns a string representation of a type for users.
func TypeString(typ Type) string {
	nType, ok := typ.(*NamedType)
	if !ok {
		return typ.String()
	}
	return nType.Package().Name.Name + "." + nType.Name()
}
