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
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

var (
	invalid        = &builtinType[ir.InvalidType]{ext: ir.InvalidType{}}
	unknown        = &builtinType[ir.UnknownType]{ext: ir.UnknownType{}}
	void           = &builtinType[ir.VoidType]{ext: ir.VoidType{}}
	emptyInterface = &builtinType[ir.InterfaceType]{ext: ir.InterfaceType{}}
	float64Type    = scalarType(ir.Float64Kind)
	numberType     = &builtinType[ir.Type]{ext: (&ir.Number{}).Type()}
	boolType       = scalarType(ir.BoolKind)
	axisLengthType = &builtinType[*ir.AtomicType]{ext: ir.AxisLengthType()}
	defaultIntType = &builtinType[*ir.AtomicType]{ext: ir.DefaultIntType}
)

func invalidType() (typeNode, ir.Type) {
	return invalid, invalid.buildType()
}

func typeNodeOk(typ typeNode) (typeNode, bool) {
	return typ, typ.kind() != ir.InvalidKind
}

func scalarType(knd ir.Kind) *builtinType[*ir.AtomicType] {
	return &builtinType[*ir.AtomicType]{ext: &ir.AtomicType{Knd: knd}}
}

// builtinType encapsulates a type defined in the intermediate representation.
type builtinType[T ir.Type] struct {
	ext T
}

var (
	_ concreteTypeNode = (*builtinType[*ir.AtomicType])(nil)
	_ arrayTypeNode    = (*builtinType[*ir.AtomicType])(nil)
)

func findBuilderPackage(scope scoper, irpkg *ir.Package) *bPackage {
	pkg := scope.block().pkg()
	if &pkg.repr == irpkg {
		return pkg
	}
	pkg, err := pkg.builder().importPath(irpkg.FullName())
	if err != nil {
		return nil
	}
	return pkg
}

func toTypeNode(scope scoper, irType ir.Type) (typeNode, bool) {
	if irType == nil {
		scope.err().Append(errors.Errorf("cannot import a nil type"))
		return invalid, false
	}
	switch tpT := irType.(type) {
	case *ir.StructType:
		return importStructType(scope, tpT)
	case *ir.AtomicType:
		return &builtinType[*ir.AtomicType]{ext: tpT}, true
	case *ir.BuiltinType:
		return &builtinType[*ir.BuiltinType]{ext: tpT}, true
	case *ir.ArrayType:
		return importArrayType(scope, tpT)
	case *ir.SliceType:
		return importSliceType(scope, tpT)
	case *ir.NamedType:
		typePackage := tpT.Package()
		pkg := findBuilderPackage(scope, typePackage)
		if pkg == nil {
			scope.err().Append(errors.Errorf("package %s not registered", typePackage))
			return invalid, false
		}
		res := pkg.ns.fetchIdentNode(tpT.Name())
		if res == nil {
			scope.err().Append(errors.Errorf("cannot import type %s: name not registered", tpT.Name()))
			return invalid, false
		}
		_, typ, ok := res.typeF(scope)
		return typ, ok
	}
	return &builtinType[ir.Type]{ext: irType}, true
}

func (n *builtinType[T]) buildType() ir.Type {
	return n.ext
}

func (n *builtinType[T]) kind() ir.Kind {
	return n.ext.Kind()
}

var scalarRank = &rank{}

func (n *builtinType[T]) rank() rankNode {
	return scalarRank
}

func (n *builtinType[T]) dtype() typeNode {
	return n
}

func (n *builtinType[T]) convertTo(scope scoper, pos nodePos, dstN typeNode) (typeNode, bool) {
	dst, ok := dstN.(arrayTypeNode)
	if !ok {
		return invalid, scope.err().AppendInternalf(pos.source(), "cannot convert array to type %T", dst)
	}
	return scalarType(dst.dtype().kind()), true
}

func (n *builtinType[T]) reconcileWith(scope scoper, pos nodePos, typ typeNode) (typeNode, bool) {
	if n.kind() == ir.NumberKind {
		return typ, true
	}
	return n, true
}

func (n *builtinType[T]) isGeneric() bool {
	return false
}

func (n *builtinType[T]) String() string {
	return n.buildType().String()
}

func (n *builtinType[T]) resolveConcreteType(scope scoper) (typeNode, bool) {
	return typeNodeOk(n)
}

// namedType is a node representing a named type declaration in GX source code.
type namedType struct {
	repr ir.NamedType

	ref          typeNode
	file         *file
	methods      map[ir.Func]function
	nameToMethod map[string]function
}

var (
	_ selector         = (*namedType)(nil)
	_ concreteTypeNode = (*namedType)(nil)
)

func processTypeDecl(scope *scopeFile, decl *ast.GenDecl) bool {
	ok := true
	for _, spec := range decl.Specs {
		ok = processType(scope, spec.(*ast.TypeSpec)) && ok
	}
	return ok
}

func processType(scope *scopeFile, spec *ast.TypeSpec) bool {
	n := &namedType{
		repr: ir.NamedType{
			NameT: spec.Name.Name,
			Src:   spec,
		},
		file:         scope.file(),
		methods:      make(map[ir.Func]function),
		nameToMethod: make(map[string]function),
	}
	var ok bool
	n.ref, ok = processTypeExpr(scope, spec.Type)
	if !ok {
		return false
	}
	return scope.file().declareType(scope, spec.Name, n) && ok
}

func importType(scope *scopeFile, typ *ir.NamedType) (*namedType, bool) {
	n := &namedType{
		repr:         *typ,
		file:         scope.file(),
		methods:      make(map[ir.Func]function),
		nameToMethod: make(map[string]function),
	}
	ok := scope.file().declareType(scope, n.repr.Src.Name, n)
	if !ok {
		return n, false
	}
	if n.repr.Underlying == nil {
		return nil, scope.err().Append(errors.Errorf("cannot import type %s in package %s: underlying type is nil", typ.NameT, scope.pkg().repr.FullName()))
	}
	n.ref, ok = toTypeNode(scope, n.repr.Underlying)
	return n, ok
}

func (n *namedType) registerMethod(method ir.Func, fn function) {
	n.methods[method] = fn
	n.nameToMethod[method.Name()] = fn
	n.repr.Methods = append(n.repr.Methods, method)
}

func (n *namedType) importMethods(block *scopeFile, methods []ir.Func) {
	for _, method := range methods {
		fn := importFuncBuiltin(block, method.(*ir.FuncBuiltin))
		n.registerMethod(fn.irFunc(), fn)
	}
}

func (n *namedType) source() ast.Node {
	return n.repr.Src
}

func (n *namedType) buildType() ir.Type {
	return n.buildNamedType()
}

func (n *namedType) buildNamedType() *ir.NamedType {
	sort.Slice(n.repr.Methods, func(i, j int) bool {
		iName := n.repr.Methods[i].Name()
		jName := n.repr.Methods[j].Name()
		return iName < jName
	})
	return &n.repr
}

func (n *namedType) convertibleTo(scope scoper, typ typeNode) (bool, error) {
	return n.buildType().ConvertibleTo(scope.evalFetcher(), typ.buildType())
}

func (n *namedType) isGeneric() bool {
	return false
}

func (n *namedType) kind() ir.Kind {
	return n.repr.Kind()
}

func (n *namedType) String() string {
	return n.buildType().String()
}

func funcPos(scope scoper, fn function) string {
	fnPos, ok := fn.(interface{ source() ast.Node })
	if !ok {
		return "as a builtin"
	}
	return fmterr.PosString(scope.err().FSet().FSet, fnPos.source().Pos())
}

func (n *namedType) assignMethod(scope scoper, fn *funcDecl) bool {
	name := fn.name().Name
	if !ir.ValidName(name) {
		return true
	}
	if n.nameToMethod == nil {
		n.nameToMethod = make(map[string]function)
	}
	// Check if a method has already been defined.
	if prev, ok := n.nameToMethod[name]; ok {
		scope.err().Appendf(fn.source(), "method %s.%s already declared at %s", n.repr.Name(), name, funcPos(scope, prev))
		return false
	}
	// Check if a field with the same name has already been defined.
	structType, ok := n.ref.(*structType)
	if ok {
		if _, defined := structType.nameToField[name]; defined {
			scope.err().Appendf(fn.source(), "field and method with the same name %s", name)
			return false
		}
	}
	n.registerMethod(&fn.ext, fn)
	return true
}

func (n *namedType) resolveConcreteType(scope scoper) (typeNode, bool) {
	if n.repr.Underlying != nil {
		return typeNodeOk(n)
	}
	underlying, ok := resolveType(scope, n, n.ref)
	if !ok {
		_, n.repr.Underlying = invalidType()
		return typeNodeOk(n)
	}
	n.repr.Underlying = underlying.buildType()
	return n, true
}

func (n *namedType) buildSelectNode(scope scoper, expr *ast.SelectorExpr) selectNode {
	if fn, ok := n.nameToMethod[expr.Sel.Name]; ok {
		return buildMethodSelectorExpr(expr, n, fn.irFunc())
	}
	underlying, ok := n.ref.(selector)
	if !ok {
		return nil
	}
	node := underlying.buildSelectNode(scope, expr)
	if node == nil {
		scope.err().Appendf(expr, "%s.%s undefined (type %s has no field or method %s)", n.repr.Name(), expr.Sel.Name, n.repr.Name(), expr.Sel.Name)
		return node
	}
	return node
}

// tupleType is a type representing the return of a function.
// If a function returns a single value, the type of the return
// will be the type of the single value. This type is only used
// when a function returns more than one value.
type tupleType struct {
	fn     *funcType
	fields []*field
}

var _ typeNode = (*tupleType)(nil)

func toTupleType(fn *funcType, fields *fieldList) *tupleType {
	return &tupleType{fn: fn, fields: fields.fields()}
}

func (tupleType) buildType() ir.Type {
	return nil
}

func (tupleType) kind() ir.Kind {
	return tupleKind
}

func (tupleType) isGeneric() bool {
	return false
}

func (n tupleType) elt(i int) *field {
	return n.fields[i]
}

func (n tupleType) len() int {
	return len(n.fields)
}

func (n *tupleType) types() []typeNode {
	typs := make([]typeNode, len(n.fields))
	for i, field := range n.fields {
		typs[i] = field.typ()
	}
	return typs
}

func (n *tupleType) String() string {
	all := make([]string, n.len())
	for i, field := range n.fields {
		all[i] = field.group.typ.String()
	}
	return "(" + strings.Join(all, ",") + ")"
}

func resolveType(scope scoper, src nodePos, typ typeNode) (out typeNode, ok bool) {
	concrete, ok := typ.(concreteTypeNode)
	if !ok {
		scope.err().AppendInternalf(src.source(), "%T is not a concrete type", typ)
		return typeNodeOk(invalid)
	}
	return concrete.resolveConcreteType(scope)
}

func resolveGenericCallType(scope scoper, calleeName string, src ast.Node, typ typeNode, fetcher ir.Fetcher, call *ir.CallExpr) (out *funcType, ok bool) {
	if calleeName == einsum {
		scope.err().Appendf(src, "%s can only be called in an assignment statement", calleeName)
		return nil, false
	}
	generic, ok := typ.(genericCallTypeNode)
	if !ok {
		scope.err().Appendf(src, "cannot call non-generic call %s (of kind %s)", calleeName, nodeKindS(typ))
		return nil, false
	}
	return generic.resolveGenericCallType(scope, src, fetcher, call)
}

func findUnderlying(typ typeNode) typeNode {
	for ok := true; ok; {
		switch tpT := typ.(type) {
		case *namedType:
			typ = tpT.ref
		default:
			ok = false
		}
	}
	return typ
}

type toTypeCaster interface {
	toTypeCast() *typeCast
}

func toTypeCast(expr exprNode) *typeCast {
	caster, ok := expr.(toTypeCaster)
	if !ok {
		return nil
	}
	return caster.toTypeCast()
}
