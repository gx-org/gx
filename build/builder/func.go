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
	"fmt"
	"go/ast"

	"github.com/gx-org/gx/build/ir"
)

type funcType struct {
	ext      *ir.FuncType
	receiver *namedType
	params   *fieldList
	results  *fieldList
}

func processFuncType(owner owner, src *ast.FuncType, decl *funcDecl) (*funcType, bool) {
	n := &funcType{
		ext: &ir.FuncType{
			Src: src,
		},
	}
	var paramsOk bool
	n.params, paramsOk = processFieldList(owner, src.Params, decl.assignArgField)
	var resultsOk bool
	n.results, resultsOk = processFieldList(owner, src.Results, decl.assignResultField)
	return n, paramsOk && resultsOk
}

func importFuncType(scope scoper, ext *ir.FuncType) (*funcType, bool) {
	n := &funcType{ext: ext}
	paramsOk, resultsOk := true, true
	if ext != nil {
		n.params, paramsOk = importFieldList(scope, ext.Params)
		n.results, resultsOk = importFieldList(scope, ext.Results)
	}
	return n, paramsOk && resultsOk
}

func importFuncBuiltinType(scope scoper, src ast.Node, fn *ir.FuncBuiltin, fetcher ir.Fetcher, call *ir.CallExpr) (*funcType, bool) {
	ftype, err := fn.Impl.BuildFuncType(fetcher, call)
	if err != nil {
		scope.err().Append(err)
		return nil, false
	}
	return importFuncType(scope, ftype)
}

func (n *funcType) source() ast.Node {
	return n.ext.Src
}

func (n *funcType) buildFuncType() *ir.FuncType {
	n.ext.Params = n.params.buildType()
	n.ext.Results = n.results.buildType()
	if n.receiver != nil {
		n.ext.Receiver = &n.receiver.repr
	}
	return n.ext
}

func (n *funcType) buildType() ir.Type {
	return n.buildFuncType()
}

func (n *funcType) convertibleTo(scope scoper, typ typeNode) (bool, error) {
	return n.buildType().ConvertibleTo(scope.evalFetcher(), typ.buildType())
}

func (n *funcType) kind() ir.Kind {
	return ir.FuncKind
}

func (n *funcType) isGeneric() bool {
	return false
}

func (n *funcType) String() string {
	results := n.results.String()
	if n.results.numFields() > 1 {
		results = fmt.Sprintf("(%s)", results)
	}
	return fmt.Sprintf("func(%s) %s", n.params.String(), results)
}

// resultTypes returns a slice of the GX types returned by the function.
func (n *funcType) resultTypes() *tupleType {
	return toTupleType(n, n.results)
}

func (n *funcType) resolveType(scope scoper) (typeNode, bool) {
	var paramsOk, resultsOk bool
	n.params, paramsOk = n.params.resolveType(scope)
	n.results, resultsOk = n.results.resolveType(scope)
	return n, paramsOk && resultsOk
}

func (n *funcType) resolveGenericCallType(scope scoper, src ast.Node, _ ir.Fetcher, call *ir.CallExpr) (*funcType, bool) {
	typ := funcType{
		ext:      &ir.FuncType{Src: n.ext.Src},
		receiver: n.receiver,
	}

	var paramsOk, resultsOk bool
	typ.params, paramsOk = n.params.resolveType(scope)
	typ.results, resultsOk = n.results.resolveType(scope)
	if !paramsOk || !resultsOk {
		return &typ, false
	}
	return &typ, paramsOk && resultsOk
}

type (
	function interface {
		typeNode() typeNode
		name() *ast.Ident
		irFunc() ir.Func
	}

	funcDecl struct {
		bFile *file
		ext   ir.FuncDecl

		ns       *namespace
		body     *blockStmt
		recv     *fieldList
		funcType *funcType

		checkFirstResultWithReceiver bool

		processOk, resolveTypeOk bool
	}
)

var (
	_ genericCallTypeNode = (*funcDecl)(nil)
	_ function            = (*funcDecl)(nil)
)

func (bFile *file) processFunc(fileScope *scopeFile, fn *ast.FuncDecl) bool {
	f := &funcDecl{
		bFile: bFile,
		ext: ir.FuncDecl{
			Src:   fn,
			FFile: &bFile.repr,
		},
	}
	f.ns = bFile.ns.newChild()
	scope := fileScope.scopeFunc(f, f.ns)
	var recvOk bool
	f.recv, recvOk = processFieldList(scope, fn.Recv, f.assignArgField)
	if f.recv != nil && f.recv.ext.Src.NumFields() > 1 {
		scope.err().Appendf(f.source(), "method has multiple receivers")
	}
	var typeOk bool
	f.funcType, typeOk = processFuncType(scope, fn.Type, f)
	if fn.Type.Results.NumFields() == 0 {
		scope.err().Appendf(f.source(), "function %s does not return a value in its signature", f.ext.Name())
		typeOk = false
	}
	var bodyOk bool
	f.body, bodyOk = processBlockStmt(scope, f, f.ext.Src.Body)
	f.processOk = typeOk && recvOk && bodyOk
	declarationOk := true
	if f.recv == nil {
		declarationOk = bFile.declareFunc(fileScope, f)
	} else {
		bFile.declareMethod(fileScope, f)
	}
	return f.processOk && declarationOk
}

func (f *funcDecl) assignArgField(block *scopeFile, fld *field) bool {
	if prev := f.ns.assign(fld.ext.Name, nil, fld.group.typ); prev != nil {
		appendRedeclaredError(block.err(), fld.ext.Name, prev)
		return false
	}
	return true
}

func (f *funcDecl) assignResultField(block *scopeFile, fld *field) (ok bool) {
	prev := f.ns.assign(fld.ext.Name, nil, fld.group.typ)
	if prev == nil {
		return true
	}
	defer func() {
		if !ok {
			appendRedeclaredError(block.err(), fld.ext.Name, prev)
		}
	}()
	if f.recv == nil ||
		len(f.recv.list) != 1 ||
		len(f.recv.list[0].ext.Fields) != 1 {
		return false
	}
	if fld.ext.Name.Name == f.recv.list[0].ext.Fields[0].Name.Name {
		f.checkFirstResultWithReceiver = true
		return true
	}
	return false
}

func (f *funcDecl) resolveReceiver(scope scoper) bool {
	if f.recv == nil {
		return true
	}
	var ok bool
	f.recv, ok = f.recv.resolveType(scope)
	if !ok {
		return false
	}
	if len(f.recv.list) != 1 {
		return false
	}
	recvType := f.recv.list[0].typ
	namedType, ok := recvType.(*namedType)
	if !ok {
		scope.err().Appendf(f.source(), "cannot define new methods on non-local type %s", recvType.String())
		return false
	}
	f.funcType.receiver = namedType
	return namedType.assignMethod(scope, f)
}

func (f *funcDecl) checkReceiverWithResult(scope scoper) bool {
	if !f.resolveTypeOk {
		return false
	}
	if !f.checkFirstResultWithReceiver {
		return true
	}
	recvType := f.funcType.receiver.buildType()
	firstResultType := f.funcType.resultTypes().elt(0).typ().buildType()
	eq, err := recvType.Equal(scope.evalFetcher(), firstResultType)
	if err != nil {
		scope.err().Append(err)
		return false
	}
	if !eq {
		scope.err().Appendf(f.source(), "first result type %s needs to equal receiver type %s because they share the same name", firstResultType.String(), recvType.String())
		return false
	}
	return true
}

func (f *funcDecl) recResolveTypes(parentScope *scopeFile) {
	if !f.processOk {
		return
	}
	scope := parentScope.scopeFunc(f, f.ns)
	f.resolveTypeOk = f.resolveReceiver(scope)
	_, funcTypeOk := f.funcType.resolveType(scope)
	recvResultOk := f.checkReceiverWithResult(scope)
	bodyOk := f.body.resolveType(scope)
	f.resolveTypeOk = funcTypeOk && f.resolveTypeOk && recvResultOk && bodyOk
	f.ext.FType = f.funcType.buildFuncType()
}

func (f *funcDecl) recBuildStmt() *ir.FuncDecl {
	if !f.resolveTypeOk {
		return &f.ext
	}
	f.ext.FType = f.funcType.buildFuncType()
	f.ext.Body = f.body.buildBlockStmt()
	return &f.ext
}

func (f *funcDecl) name() *ast.Ident {
	return f.ext.Src.Name
}

func (f *funcDecl) kind() ir.Kind {
	return ir.FuncKind
}

func (f *funcDecl) irFunc() ir.Func {
	return f.recBuildStmt()
}

func (f *funcDecl) isGeneric() bool {
	return false
}

func (f *funcDecl) buildType() ir.Type {
	return f.funcType.buildType()
}

func (f *funcDecl) typeNode() typeNode {
	return f
}

// Pos is the position of the function in the code.
func (f *funcDecl) source() ast.Node {
	return f.ext.Src
}

// String returns a string representation of the function.
func (f *funcDecl) String() string {
	return f.ext.Name()
}

func (f *funcDecl) resolveGenericCallType(scope scoper, src ast.Node, fetcher ir.Fetcher, call *ir.CallExpr) (*funcType, bool) {
	return f.funcType.resolveGenericCallType(scope, src, fetcher, call)
}

// funcBuiltin is a function imported from a package.
type funcBuiltin struct {
	ext ir.FuncBuiltin
}

var (
	_ genericCallTypeNode = (*funcBuiltin)(nil)
	_ function            = (*funcBuiltin)(nil)
)

func importFuncBuiltin(scope *scopeFile, ext *ir.FuncBuiltin) *funcBuiltin {
	return &funcBuiltin{ext: *ext}
}

func (fn *funcBuiltin) buildType() ir.Type {
	return fn.ext.Type()
}

func (fn *funcBuiltin) resolveGenericCallType(scope scoper, src ast.Node, fetcher ir.Fetcher, call *ir.CallExpr) (*funcType, bool) {
	typ, ok := importFuncBuiltinType(scope, src, &fn.ext, fetcher, call)
	if !ok {
		return typ, ok
	}
	return typ.resolveGenericCallType(scope, src, fetcher, call)
}

func (fn *funcBuiltin) irFunc() ir.Func {
	return &fn.ext
}

func (fn *funcBuiltin) name() *ast.Ident {
	return &ast.Ident{Name: fn.ext.Name()}
}

func (fn *funcBuiltin) isGeneric() bool {
	return false
}

func (fn *funcBuiltin) typeNode() typeNode {
	return fn
}

func (fn *funcBuiltin) kind() ir.Kind {
	return ir.FuncKind
}

func (fn *funcBuiltin) String() string {
	return fn.ext.File().Package.Name.Name + "." + fn.ext.Name()
}
