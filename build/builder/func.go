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
	ext        *ir.FuncType
	receiver   *namedType
	typeParams *fieldList
	params     *fieldList
	results    *fieldList
}

func processFuncType(owner owner, src *ast.FuncType, decl *funcDecl) (*funcType, bool) {
	n := &funcType{
		ext: &ir.FuncType{
			Src: src,
		},
	}
	var typesOk bool
	n.typeParams, typesOk = processFieldList(owner, src.TypeParams, decl.assignTypeField)
	var paramsOk bool
	n.params, paramsOk = processFieldList(owner, src.Params, decl.assignArgField)
	var resultsOk bool
	n.results, resultsOk = processFieldList(owner, src.Results, decl.assignResultField)
	return n, typesOk && paramsOk && resultsOk
}

func importFuncType(scope scoper, ext *ir.FuncType) (*funcType, bool) {
	n := &funcType{ext: ext}
	typesOk, paramsOk, resultsOk := true, true, true
	if ext != nil {
		n.typeParams, typesOk = importFieldList(scope, ext.TypeParams)
		n.params, paramsOk = importFieldList(scope, ext.Params)
		n.results, resultsOk = importFieldList(scope, ext.Results)
	}
	return n, typesOk && paramsOk && resultsOk
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
	n.ext.TypeParams = n.typeParams.irType()
	n.ext.Params = n.params.irType()
	n.ext.Results = n.results.irType()
	if n.receiver != nil {
		n.ext.Receiver = &n.receiver.repr
	}
	return n.ext
}

func (n *funcType) irType() ir.Type {
	return n.buildFuncType()
}

func (n *funcType) convertibleTo(scope scoper, typ typeNode) (bool, error) {
	return n.irType().ConvertibleTo(scope.evalFetcher(), typ.irType())
}

func (n *funcType) kind() ir.Kind {
	return ir.FuncKind
}

func (n *funcType) isGeneric() bool {
	return false
}

func (n *funcType) String() string {
	typeParams := ""
	if !n.typeParams.empty() {
		typeParams = fmt.Sprintf("[%s]", n.typeParams.String())
	}

	results := n.results.String()
	if n.results.numFields() > 1 {
		results = fmt.Sprintf("(%s)", results)
	}
	return fmt.Sprintf("func%s(%s) %s", typeParams, n.params.String(), results)
}

// resultTypes returns a slice of the GX types returned by the function.
func (n *funcType) resultTypes() *tupleType {
	return toTupleType(n, n.results)
}

func (n *funcType) resolveType(scope scoper) (typeNode, bool) {
	receiverOk := true
	if n.receiver != nil {
		_, receiverOk = n.receiver.resolveConcreteType(scope)
	}
	var paramsOk, resultsOk, typesOk bool
	n.typeParams, typesOk = n.typeParams.resolveType(scope)
	n.params, paramsOk = n.params.resolveType(scope)
	n.results, resultsOk = n.results.resolveType(scope)
	return n, typesOk && paramsOk && resultsOk && receiverOk
}

func (n *funcType) resolveGenericCallType(scope scoper, _ ir.Fetcher, call *callExpr) (*funcType, bool) {
	typ := funcType{
		ext:      &ir.FuncType{Src: n.ext.Src},
		receiver: n.receiver,
	}

	numTypeParams := n.typeParams.numFields()
	if len(call.typeArgs) > numTypeParams {
		scope.err().Appendf(call.source(), "too many type arguments in call to %s", call.callee)
		return nil, false
	}
	if len(call.typeArgs) < numTypeParams {
		if len(call.typeArgs) == 0 {
			scope.err().Appendf(call.source(), "type parameter inference is not supported")
		} else {
			scope.err().Appendf(call.source(), "not enough type arguments in call to %s", call.callee)
		}
		return nil, false
	}

	for i, typeParam := range n.typeParams.fields() {
		// Type parameters may change from call to call, so wrap the concrete type in deferredType to
		// prevent caching.
		scope.namespace().assign(newIdent(typeParam.ext.Name, &deferredType{src: call.source(), ref: call.typeArgs[i]}))
	}

	var paramsOk, resultsOk bool
	typ.typeParams = n.typeParams // For now, just copy type parameters.
	typ.params, paramsOk = n.params.resolveType(scope)
	typ.results, resultsOk = n.results.resolveType(scope)
	if !paramsOk || !resultsOk {
		return &typ, false
	}
	return &typ, paramsOk && resultsOk
}

type (
	function interface {
		staticValueNode
		name() *ast.Ident
		irFunc() ir.Func
	}

	funcDecl struct {
		bFile *file
		ext   ir.FuncDecl

		ns       *blockNamespace
		recv     *fieldList
		funcType *funcType

		meta *assignDirective
		body *blockStmt

		checkFirstResultWithReceiver bool

		processOk         bool
		resolveTypeStatus done
	}
)

var (
	_ genericCallTypeNode = (*funcDecl)(nil)
	_ staticValueNode     = (*funcDecl)(nil)
	_ function            = (*funcDecl)(nil)
)

func (bFile *file) processFunc(fileScope *scopeFile, fn *ast.FuncDecl) bool {
	dir, dirComment, dirOk := processFuncDirective(fileScope, fn)
	var funOk bool
	switch dir {
	case none:
		if fn.Body == nil {
			funOk = bFile.processBuiltinFunc(fileScope, fn)
		} else {
			funOk = bFile.processFuncDecl(fileScope, fn)
		}
	case assign: // Function body assigned via gx:=
		funOk = bFile.processAssignFunc(fileScope, fn, dirComment)
	case irmacro: // IR Macro function that will be called by the compiler via gx:irmacro
		funOk = bFile.processIRMacroFunc(fileScope, fn, dirComment)
	default:
		return fileScope.err().AppendInternalf(dirComment, "directive %d not supported", dir)
	}
	return dirOk && funOk
}

func newFuncDecl(fileScope *scopeFile, fn *ast.FuncDecl) (*funcDecl, *scopeBlock) {
	bFile := fileScope.file()
	f := &funcDecl{
		bFile: bFile,
		ext: ir.FuncDecl{
			Src:   fn,
			FFile: &bFile.repr,
		},
		ns: bFile.ns.newChild(),
	}
	scope := f.funcScope(fileScope)
	var recvOk bool
	f.recv, recvOk = processFieldList(scope, fn.Recv, f.assignArgField)
	if f.recv != nil && f.recv.ext.Src.NumFields() > 1 {
		scope.err().Appendf(f.source(), "method has multiple receivers")
	}
	var typeOk bool
	f.funcType, typeOk = processFuncType(scope, fn.Type, f)
	f.processOk = typeOk && recvOk
	return f, scope
}

func (bFile *file) processFuncDecl(fileScope *scopeFile, fn *ast.FuncDecl) bool {
	f, scope := newFuncDecl(fileScope, fn)
	if !fileScope.file().declareFuncDecl(fileScope, f) {
		f.processOk = false
	}
	if !f.checkReturnValue(scope) {
		f.processOk = false
	}
	if !f.processOk {
		return false
	}
	f.processOk = f.processFuncDefinition(scope)
	return f.processOk
}

func (bFile *file) processAssignFunc(fileScope *scopeFile, fn *ast.FuncDecl, comment *ast.Comment) bool {
	f, scope := newFuncDecl(fileScope, fn)
	if !fileScope.file().declareFuncDecl(fileScope, f) {
		f.processOk = false
	}
	if !checkEmptyParamsResults(fileScope, fn, "assigned") {
		f.processOk = false
	}
	var ok bool
	if f.meta, ok = f.processFuncAssignDirective(scope, comment); !ok {
		f.processOk = false
	}
	return f.processOk
}

func (f *funcDecl) processFuncDefinition(scope *scopeBlock) bool {
	if !f.checkReturnValue(scope) {
		f.processOk = false
	}
	var bodyOk bool
	f.body, bodyOk = processBlockStmt(scope, f, f.ext.Src.Body)
	f.processOk = f.processOk && bodyOk
	return f.processOk
}

func (f *funcDecl) receiver() *fieldList {
	return f.recv
}

func checkEmptyParamsResults(scope *scopeFile, fn *ast.FuncDecl, errPrefix string) bool {
	ok := true
	if fn.Type.Params.NumFields() != 0 {
		ok = scope.err().Appendf(fn, "%s function has parameters", errPrefix)
	}
	if fn.Type.Results.NumFields() != 0 {
		ok = scope.err().Appendf(fn, "%s function has return values", errPrefix)
	}
	return ok
}

func (f *funcDecl) checkReturnValue(scope *scopeBlock) bool {
	if f.ext.Src.Type.Results.NumFields() == 0 {
		scope.err().Appendf(f.source(), "function %s does not return a value in its signature", f.ext.Name())
		return false
	}
	return true
}

func (f *funcDecl) assignTypeField(block *scopeFile, fld *field) bool {
	if prev := f.ns.fetch(fld.ext.Name.Name); prev != nil {
		block.err().Appendf(fld.ext.Name, "type parameter %s redeclared", fld.ext.Name.String())
		return false
	}
	f.ns.assign(newIdent(fld.ext.Name, fld.typ()))
	return true
}

func (f *funcDecl) assignArgField(block *scopeFile, fld *field) bool {
	if prev := f.ns.fetch(fld.ext.Name.Name); prev != nil {
		appendRedeclaredError(block.err(), fld.ext.Name, prev)
		return false
	}
	f.ns.assign(newIdent(fld.ext.Name, fld.group.typ))
	return true
}

func (f *funcDecl) assignResultField(block *scopeFile, fld *field) (ok bool) {
	prev := f.ns.fetch(fld.ext.Name.Name)
	if prev == nil {
		return true
	} else {
		f.ns.assign(newIdent(fld.ext.Name, fld.group.typ))
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
	if f.resolveTypeStatus == doneNotOk {
		return false
	}
	if !f.checkFirstResultWithReceiver {
		return true
	}
	if f.funcType.receiver == nil {
		return true
	}
	recvType := f.funcType.receiver.irType()
	firstResultType := f.funcType.resultTypes().elt(0).typ().irType()
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

// funcScope returns a new function scope with access to the function's definitions (e.g. both type
// and regular parameters); each returned scope is isolated from any others so that modifications to
// that scope are not visible elsewhere.
func (f *funcDecl) funcScope(parent owner) *scopeBlock {
	return parent.fileScope().scopeFunc(f, f.ns.newChild())
}

func (f *funcDecl) resolveType(parentScope scoper) (typeNode, bool) {
	if f.resolveTypeStatus.isDone() {
		return typeNodeOk(f.funcType)
	}
	if !f.processOk {
		return typeNodeOk(invalid)
	}
	scope := f.funcScope(parentScope)
	resolveTypeOk := f.resolveReceiver(scope)
	_, funcTypeOk := f.funcType.resolveType(scope)
	recvResultOk := f.checkReceiverWithResult(scope)
	f.resolveTypeStatus = toDone(funcTypeOk && resolveTypeOk && recvResultOk)
	return f.funcType, f.resolveTypeStatus.ok()
}

func (f *funcDecl) resolveBody(parentScope *scopeFile) {
	if !f.resolveTypeStatus.ok() {
		return
	}
	scope := f.funcScope(parentScope)
	if f.body == nil {
		scope.err().Appendf(f.ext.Src, "function %s has no body", f.name().Name)
		f.resolveTypeStatus = doneNotOk
		return
	}
	f.resolveTypeStatus = toDone(f.body.resolveType(scope))
	f.ext.FType = f.funcType.buildFuncType()
}

func (f *funcDecl) resolve(parentScope *scopeFile) {
	if _, ok := f.resolveType(parentScope); !ok {
		return
	}
	if f.meta == nil {
		f.resolveBody(parentScope)
	} else {
		f.buildBodyWithMacro(parentScope)
	}
	if !f.resolveTypeStatus.ok() {
		return
	}
	if f.ext.Body != nil {
		return
	}
	f.ext.FType = f.funcType.buildFuncType()
	f.ext.Body = f.body.buildBlockStmt()
}

func (f *funcDecl) buildBodyWithMacro(scope *scopeFile) {
	if !f.resolveTypeStatus.ok() {
		return
	}
	f.resolveTypeStatus = toDone(f.meta.buildIR(scope))
}

func (f *funcDecl) recBuildStmt() *ir.FuncDecl {
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

func (f *funcDecl) staticValue() ir.StaticValue {
	return &f.ext
}

func (f *funcDecl) isGeneric() bool {
	return false
}

func (f *funcDecl) irType() ir.Type {
	return f.funcType.irType()
}

// Pos is the position of the function in the code.
func (f *funcDecl) source() ast.Node {
	return f.ext.Src
}

// String returns a string representation of the function.
func (f *funcDecl) String() string {
	return f.ext.Name()
}

func (f *funcDecl) resolveGenericCallType(scope scoper, fetcher ir.Fetcher, call *callExpr) (*funcType, bool) {
	return f.funcType.resolveGenericCallType(f.funcScope(scope), fetcher, call)
}

// funcBuiltin is a function imported from a package.
type funcBuiltin struct {
	ext ir.FuncBuiltin

	funcDecl *funcDecl
}

var (
	_ genericCallTypeNode = (*funcBuiltin)(nil)
	_ function            = (*funcBuiltin)(nil)
)

func importFuncBuiltin(scope *scopeFile, ext *ir.FuncBuiltin) (*funcBuiltin, bool) {
	fn := &funcBuiltin{ext: *ext}
	fn.ext.Package = &scope.pkg().repr
	return fn, true
}

func (bFile *file) processBuiltinFunc(fileScope *scopeFile, fn *ast.FuncDecl) bool {
	f, scope := newFuncDecl(fileScope, fn)
	if !f.checkReturnValue(scope) {
		f.processOk = false
	}
	builtin, builtinOk := importFuncBuiltin(fileScope, &ir.FuncBuiltin{
		FName: f.ext.Src.Name.String(),
		FType: f.funcType.buildFuncType(),
	})
	builtin.funcDecl = f
	declareOk := fileScope.file().declareFuncBuiltin(fileScope, builtin)
	return f.processOk && builtinOk && declareOk
}

func (fn *funcBuiltin) irType() ir.Type {
	return fn.ext.Type()
}

func (fn *funcBuiltin) resolveGenericCallType(scope scoper, fetcher ir.Fetcher, call *callExpr) (*funcType, bool) {
	if fn.ext.Impl == nil {
		scope.err().Appendf(call.source(), "builtin function %s has no implementation", fn.ext.Name())
		return nil, false
	}
	if fn.funcDecl != nil {
		return fn.funcDecl.resolveGenericCallType(scope, fetcher, call)
	}

	irCall := call.buildExpr().(*ir.CallExpr)
	typ, ok := importFuncBuiltinType(scope, call.source(), &fn.ext, fetcher, irCall)
	if !ok {
		return typ, ok
	}
	return typ.resolveGenericCallType(scope, fetcher, call)
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

func (fn *funcBuiltin) receiver() *fieldList {
	return nil
}

func (fn *funcBuiltin) staticValue() ir.StaticValue {
	return &fn.ext
}

func (fn *funcBuiltin) resolveType(scoper) (typeNode, bool) {
	return fn, true
}

func (fn *funcBuiltin) kind() ir.Kind {
	return ir.FuncKind
}

func (fn *funcBuiltin) String() string {
	return fn.ext.File().Package.Name.Name + "." + fn.ext.Name()
}

type funcLiteral struct {
	ext   *ir.FuncLit
	fdecl *funcDecl
	ftype *funcType
	body  *blockStmt
}

var _ exprNode = (*funcLiteral)(nil)

func (fn *funcLiteral) resolveType(scope scoper) (typeNode, bool) {
	ftype, ftypeOk := fn.ftype.resolveType(scope)
	bodyOk := fn.body.resolveType(fn.fdecl.funcScope(scope))
	return ftype, ftypeOk && bodyOk
}

func (fn *funcLiteral) expr() ast.Expr {
	return fn.ext.Src
}

func (fn *funcLiteral) source() ast.Node {
	return fn.ext.Src
}

func (fn *funcLiteral) buildExpr() ir.Expr {
	fn.ext.FFile = &fn.fdecl.bFile.repr
	fn.ext.FType = fn.ftype.buildFuncType()
	fn.ext.Body = fn.body.buildBlockStmt()
	return fn.ext
}

func (fn *funcLiteral) String() string {
	return "func {...}"
}

func processFuncLit(owner owner, src *ast.FuncLit) (*funcLiteral, bool) {
	fdecl := &funcDecl{
		bFile: owner.fileScope().file(),
		ns:    owner.fileScope().namespace().newChild(),
		ext:   ir.FuncDecl{},
	}
	scope := fdecl.funcScope(owner)
	ftype, ftypeOk := processFuncType(scope, src.Type, fdecl)
	body, bodyOk := processBlockStmt(scope, fdecl, src.Body)
	fdecl.funcType = ftype
	return &funcLiteral{
		ext:   &ir.FuncLit{Src: src},
		fdecl: fdecl,
		ftype: ftype,
		body:  body,
	}, ftypeOk && bodyOk
}

type funcBuilder struct {
	fn function
}

var _ irBuilder = funcBuilder{}

func (b funcBuilder) buildIR(pkg *ir.Package) {
	pkg.Funcs = append(pkg.Funcs, b.fn.irFunc())
}

type funcMacro struct {
	ext       ir.FuncMeta
	processOk bool
	ftype     *funcType
}

var _ function = (*funcMacro)(nil)

func fieldListAtomicIR() *ir.FieldList {
	group := &ir.FieldGroup{
		Type: ir.TypeFromKind(ir.IRKind),
	}
	group.Fields = []*ir.Field{&ir.Field{
		Group: group,
	}}
	return &ir.FieldList{
		List: []*ir.FieldGroup{group},
	}
}

func (bFile *file) processIRMacroFunc(scope *scopeFile, fn *ast.FuncDecl, comment *ast.Comment) bool {
	f := &funcMacro{
		ext: ir.FuncMeta{
			Src:   fn,
			FFile: &bFile.repr,
			FType: &ir.FuncType{
				Params:  fieldListAtomicIR(),
				Results: fieldListAtomicIR(),
			},
		},
	}
	f.processOk = checkEmptyParamsResults(scope, fn, "irmacro")
	if fn.Recv.NumFields() != 0 {
		f.processOk = scope.err().Appendf(fn, "irmacro function has a receiver")
	}
	var importOk bool
	f.ftype, importOk = importFuncType(scope, f.ext.FType)
	if !importOk {
		f.processOk = false
	}
	if !bFile.declareFuncMacro(scope, f) {
		f.processOk = false
	}
	return f.processOk
}

func (fn *funcMacro) staticValue() ir.StaticValue {
	return &fn.ext
}

func (fn *funcMacro) resolveType(scope scoper) (typeNode, bool) {
	return unknown, true
}

func (fn *funcMacro) name() *ast.Ident {
	return fn.ext.Src.Name
}

func (fn *funcMacro) irFunc() ir.Func {
	return &fn.ext
}
