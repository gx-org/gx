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

package testing

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

// Visitor checks node in the intermediate representation.
type Visitor func(errs *fmterr.Errors, node ir.Node)

type validator struct {
	errs     fmterr.Errors
	visited  map[ir.Node]bool
	visitors []Visitor
}

// Validate an intermediate representation tree to make sure all fields are set.
func Validate(node ir.Node, visitors ...Visitor) error {
	v := validator{
		visitors: visitors,
		visited:  map[ir.Node]bool{},
	}
	v.validate(node)
	return v.errs.ToError()
}

func (v *validator) validate(node ir.Node) {
	if node == nil {
		return
	}
	if reflect.TypeOf(node).Kind() == reflect.Pointer && reflect.ValueOf(node).IsNil() {
		return
	}
	if v.visited[node] {
		return
	}
	v.visited[node] = true

	for _, visitor := range v.visitors {
		visitor(&v.errs, node)
	}
	switch nodeT := node.(type) {

	// Declarations
	case *ir.Package:
		v.validatePackage(nodeT)
	case *ir.File:
		for _, imp := range nodeT.Imports {
			v.validate(imp)
		}
	case *ir.ConstDecl:
		for _, expr := range nodeT.Exprs {
			v.validate(expr)
		}
	case *ir.ConstExpr:
		v.validate(nodeT.Val)
	case *ir.FuncBuiltin:
		if nodeT.FType != nil {
			v.validate(nodeT.FType)
		}
	case *ir.FuncDecl:
		v.validate(nodeT.FType)
		if nodeT.Body == nil {
			v.errs.Append(errors.Errorf("function %q has no body", nodeT.Name()))
			return
		}
		v.validate(nodeT.Body)
	case *ir.Macro:
	case *ir.FuncLit:
		v.validate(nodeT.FType)
		if nodeT.Body == nil {
			v.errs.Append(errors.Errorf("function literal has no body"))
			return
		}
		v.validate(nodeT.Body)
	case *ir.VarDecl:
		v.validate(nodeT.TypeV)
	case *ir.LocalVarStorage:
		v.validate(nodeT.Typ)
	case *ir.ImportDecl:

	// Fields
	case *ir.FieldList:
		for _, fieldGroup := range nodeT.List {
			v.validate(fieldGroup)
		}
	case *ir.FieldGroup:
		v.validate(nodeT.Type)
		for _, field := range nodeT.Fields {
			v.validate(field)
		}
	case *ir.Field:
	case *ir.FieldLit:
		v.validate(nodeT.Field)
		v.validate(nodeT.X)
	case *ir.StructFieldStorage:
		v.validate(nodeT.Sel)
	// Types
	case ir.ArrayType:
		v.validate(nodeT.DataType())
		v.validate(nodeT.Rank())
	case *ir.FuncType:
		v.validate(nodeT.Params)
		v.validate(nodeT.Results)
		if nodeT.Receiver != nil {
			v.validate(nodeT.Receiver)
		}
	case *ir.NamedType:
		v.validate(nodeT.Underlying)
	case *ir.SliceType:
		v.validate(nodeT.DType)
	case *ir.StructType:
		v.validate(nodeT.Fields)

	// Dimensions
	case *ir.Rank:
		for _, axis := range nodeT.Ax {
			v.validate(axis)
		}
	case *ir.RankInfer:
		if nodeT.Rnk != nil {
			v.validate(nodeT.Rnk)
		}
	case *ir.AxisExpr:
		v.validate(nodeT.X)
	case *ir.AxisInfer:
		v.validate(nodeT.X)

	// Expressions:
	case *ir.ArrayLitExpr:
		for _, val := range nodeT.Values() {
			v.validate(val)
		}
	case *ir.BinaryExpr:
		v.validate(nodeT.Typ)
		v.validate(nodeT.X)
		v.validate(nodeT.Y)
	case *ir.FuncValExpr:
		v.validate(nodeT.X)
		v.validate(nodeT.F)
		v.validate(nodeT.T)
	case *ir.CallExpr:
		v.validate(nodeT.Callee)
		for _, arg := range nodeT.Args {
			v.validate(arg)
		}
	case *ir.CastExpr:
		v.validate(nodeT.Typ)
		v.validate(nodeT.X)
	case *ir.TypeAssertExpr:
		v.validate(nodeT.Typ)
		v.validate(nodeT.X)
	case *ir.SelectorExpr:
		v.validate(nodeT.Stor)
		v.validate(nodeT.X)
	case *ir.IndexExpr:
		v.validate(nodeT.X)
		v.validate(nodeT.Index)
		v.validate(nodeT.Typ)
	case *ir.EinsumExpr:
		v.validate(nodeT.X)
		v.validate(nodeT.Y)
		v.validate(nodeT.Typ)
	case *ir.PackageRef:
		v.validate(nodeT.Decl)
	case *ir.ParenExpr:
		v.validate(nodeT.X)
	case *ir.NumberInt:
	case *ir.NumberFloat:
	case *ir.NumberCastExpr:
		v.validate(nodeT.X)
		v.validate(nodeT.Typ)
	case *ir.SliceLitExpr:
		for _, expr := range nodeT.Elts {
			v.validate(expr)
		}
	case *ir.StructLitExpr:
		v.validate(nodeT.Typ)
		for _, elt := range nodeT.Elts {
			v.validate(elt)
		}
	case *ir.UnaryExpr:
		v.validate(nodeT.X)
	case *ir.ValueRef:
		v.validate(nodeT.Stor)
	case *ir.TypeValExpr:
		v.validate(nodeT.X)
		v.validate(nodeT.Typ)

	// Statements
	case *ir.ReturnStmt:
		for _, res := range nodeT.Results {
			v.validate(res)
		}
	case *ir.AssignExprStmt:
		for _, assign := range nodeT.List {
			v.validate(assign.X)
			v.validate(assign.Storage)
		}
	case *ir.AssignCallStmt:
		v.validate(nodeT.Call)
		for _, assign := range nodeT.List {
			v.validate(assign)
		}
	case *ir.RangeStmt:
		v.validate(nodeT.Key)
		v.validate(nodeT.Value)
		v.validate(nodeT.X)
		for _, stmt := range nodeT.Body.List {
			v.validate(stmt)
		}
	case *ir.IfStmt:
		v.validate(nodeT.Init)
		v.validate(nodeT.Cond)
		for _, stmt := range nodeT.Body.List {
			v.validate(stmt)
		}
		v.validate(nodeT.Else)
	case *ir.BlockStmt:
		for _, stmt := range nodeT.List {
			v.validate(stmt)
		}
	case *ir.ExprStmt:
		v.validate(nodeT.X)
	case *ir.BuiltinType:
		if nodeT.Impl == nil {
			v.errs.Append(errors.Errorf("builtin has no implementation"))
		}
	case ir.Type:
	default:
		if !ir.IsExported(reflect.TypeOf(node).Name()) {
			return
		}
		v.errs.Append(errors.Errorf("type %T not supported by gxtesting.Validate", nodeT))
	}
}

type checkUnique struct {
	v        *validator
	declared map[string]string
}

func (c *checkUnique) checkName(kind string, n string) {
	if prev := c.declared[n]; prev != "" {
		c.v.errs.Append(errors.Errorf("%s %s has already been declared as a %s", kind, n, prev))
		return
	}
	c.declared[n] = kind
}

func (v *validator) validatePackage(pkg *ir.Package) {
	for _, file := range pkg.Files {
		v.validate(file)
	}
	unique := checkUnique{v: v, declared: make(map[string]string)}
	for _, cst := range pkg.Decls.Consts {
		for _, expr := range cst.Exprs {
			unique.checkName("constant", expr.VName.Name)
		}
		v.validate(cst)
	}
	for _, fct := range pkg.Decls.Funcs {
		unique.checkName("function", fct.Name())
		v.validate(fct)
	}
	for _, typ := range pkg.Decls.Types {
		unique.checkName("type", typ.Name())
		v.validate(typ)
	}
	for _, vr := range pkg.Decls.Vars {
		for _, expr := range vr.Exprs {
			unique.checkName("variable", expr.VName.Name)
			v.validate(expr)
		}
	}
}

// CheckSource checks that Source and File return non-nil values.
func CheckSource(errs *fmterr.Errors, node ir.Node) {
	src, ok := node.(ir.SourceNode)
	if !ok {
		return
	}
	if src.Source() == nil {
		errs.Append(fmt.Errorf("validator: %T.Source() returns nil", src))
	}
}
