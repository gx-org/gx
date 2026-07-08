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

	"github.com/gx-org/gx/base/ordered"
	"github.com/gx-org/gx/build/builder/irb"
	"github.com/gx-org/gx/build/ir"
)

type signatureNamespace struct {
	fType *funcType
	names map[string]*field
}

func (ns *signatureNamespace) assignTypeField(pscope procScope, fld *field) bool {
	if prev := ns.names[fld.src.Name]; prev != nil {
		pscope.Err().Appendf(fld.src, "type parameter %s redeclared", fld.src.Name)
		return false
	}
	ns.names[fld.src.Name] = fld
	return true
}

func (ns *signatureNamespace) assignField(pscope procScope, fld *field) bool {
	if prev := ns.names[fld.src.Name]; prev != nil {
		return appendRedeclaredError(pscope.Err(), fld.src.Name, prev.src, fld.src)
	}
	ns.names[fld.src.Name] = fld
	return true
}

func (ns *signatureNamespace) assignResultField(pscope procScope, fld *field) bool {
	// TODO(b/418153202): check that the types are the same.
	ns.names[fld.src.Name] = fld
	return true
}

type funcType struct {
	src      *ast.FuncType
	compEval bool

	receiver *namedType

	typeParams *fieldList
	genShapes  *ordered.Map[string, *processNodeT[*defineAxisLength]]

	recv    *fieldList
	params  *fieldList
	results *fieldList

	varargs *varargsType

	namedResults bool
}

func processFuncType(pscope procScope, src *ast.FuncType, recv *ast.FieldList, compEval bool) (*funcType, bool) {
	n := &funcType{
		src:       src,
		compEval:  compEval,
		genShapes: ordered.NewMap[string, *processNodeT[*defineAxisLength]](),
	}
	var recvOk, typesOk, paramsOk, resultsOk bool
	sig := &signatureNamespace{fType: n, names: make(map[string]*field)}
	n.recv, recvOk = processFieldList(
		defaultTypeProcScope(pscope),
		recv,
		sig.assignField,
	)
	if n.recv != nil && n.recv.numFields() > 1 {
		recvOk = pscope.Err().Appendf(recv, "method has multiple receivers")
	}
	n.typeParams, typesOk = processFieldList(
		defaultTypeProcScope(pscope),
		src.TypeParams,
		sig.assignTypeField,
	)
	n.params, paramsOk = processFieldList(
		&funcParamScope{
			defaultAxLenTypeScope: &defaultAxLenTypeScope{
				procScope: pscope,
			},
			ftype: n,
		},
		src.Params,
		sig.assignField,
	)
	n.results, resultsOk = processFieldList(
		defaultTypeProcScope(pscope),
		src.Results,
		sig.assignResultField,
	)
	return n, recvOk && typesOk && paramsOk && resultsOk
}

func rankInferOk(rscope resolveScope, src ast.Node, typ ir.Type) bool {
	array, isArray := typ.(ir.ArrayType)
	if !isArray {
		return true
	}

	if _, isInfered := array.Rank().(*ir.RankInfer); isInfered {
		return rscope.Err().Appendf(src, "cannot use an inferred rank in fields")
	}
	return true
}

func defineTypeParam(s resolveScope, storage ir.Storage) bool {
	fieldStorage := storage.(*ir.FieldStorage)
	var generic ir.GenericParam
	if ir.IsNonTypeGeneric(fieldStorage.Type()) {
		generic = ir.NewGenericNonTypeParam(fieldStorage.Field)
	} else {
		generic = ir.NewGenericTypeParam(fieldStorage.Field)
	}
	return defineLocalVar(s, generic)
}

func (n *funcType) buildFuncType(rscope resolveScope) (*ir.FuncType, *funcResolveScope, bool) {
	ext := &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: n.src},
		CompEval: n.compEval,
	}
	var tParamsOk, recvOk, paramsOk, resultsOk bool
	sigscope, ephemeralOk := newEphemeralResolveScope(rscope, n.src)
	typeParamsScope := newDefineScope(sigscope, defineTypeParam)
	ext.TypeParams, tParamsOk = n.typeParams.buildFieldList(typeParamsScope)
	ext.Receiver, recvOk = n.recv.buildFieldList(newDefineScope(sigscope, nil))
	if recvOk && ext.Receiver != nil {
		if field := ext.ReceiverField(); field.Name != nil {
			defineLocalVar(sigscope, field.Storage())
		}
	}
	paramScope := newDefineScope(sigscope, defineLocalVar)
	paramScope.ftype = ext
	ext.Params, paramsOk = n.params.buildFieldList(paramScope)
	if n.varargs != nil {
		if params := ext.Params.Fields(); len(params) > 0 {
			ext.VarArgs = params[len(params)-1].Type().(*ir.VarArgsType)
		}
	}
	ext.GenericValues = make([]ir.GenericValue, ext.TypeParams.Len())
	resultScope := newDefineScope(sigscope, nil)
	ext.Results, resultsOk = n.results.buildFieldList(resultScope)
	if resultsOk {
		for _, field := range ext.Results.Fields() {
			if !rankInferOk(rscope, field.Type().Node(), field.Type()) {
				resultsOk = false
			}
		}
	}
	fnscope, fnscopeOk := newFuncScope(rscope, ext)
	return ext, fnscope, tParamsOk && paramsOk && resultsOk && recvOk && fnscopeOk && ephemeralOk
}

func (n *funcType) source() ast.Node {
	return n.src
}

func (n *funcType) buildTypeExpr(rscope resolveScope) (*ir.TypeValExpr, bool) {
	tp, _, ok := n.buildFuncType(rscope)
	if !ok {
		return nil, false
	}
	return ir.TypeExpr(tp, tp), true
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

type funcDecl struct {
	bFile *file
	src   *ast.FuncDecl
	fType *funcType
	body  *blockStmt
}

var _ function = (*funcDecl)(nil)

func (bFile *file) processFunc(fileScope procScope, src *ast.FuncDecl) bool {
	dir, dirComment, dirOk := processFuncAttribute(fileScope, src)
	var fn function
	var ok bool
	switch dir {
	case none:
		fn, ok = bFile.processDeclaredFunc(fileScope, src, false)
	case irmacro: // IR Macro function that will be called by the compiler via gx:irmacro
		fn, ok = bFile.processIRMacroFunc(fileScope, src, dirComment)
	case cpeval:
		fn, ok = bFile.processDeclaredFunc(fileScope, src, true)
	case funcAnnotator:
		fn, ok = bFile.processAnnotatorFunc(fileScope, src, dirComment)
	case fieldAnnotator:
		fn, ok = bFile.processAnnotatorField(fileScope, src, dirComment)
	default:
		return fileScope.Err().AppendInternalf(dirComment, "directive %d not supported", dir)
	}
	if !ok {
		return false
	}
	fn, ok = processFuncAnnotations(fileScope, src, fn)
	if !ok {
		return false
	}
	_, ok = fileScope.decls().registerFunc(fn)
	return dirOk && ok
}

func (bFile *file) processDeclaredFunc(fileScope procScope, src *ast.FuncDecl, compEval bool) (function, bool) {
	if src.Body == nil {
		return bFile.processBuiltinFunc(fileScope, src, compEval)
	}
	return bFile.processFuncDecl(fileScope, src, compEval)
}

func newFuncDecl(scope procScope, fn *ast.FuncDecl, compEval bool) (*funcDecl, bool) {
	f := &funcDecl{bFile: scope.file(), src: fn}
	var ok bool
	f.fType, ok = processFuncType(scope, fn.Type, fn.Recv, compEval)
	return f, ok
}

func (bFile *file) processFuncDecl(pscope procScope, src *ast.FuncDecl, compEval bool) (function, bool) {
	f, declOk := newFuncDecl(pscope, src, compEval)
	var bodyOk bool
	f.body, bodyOk = processBlockStmt(pscope, f.src.Body)
	return f, declOk && bodyOk
}

func (f *funcDecl) file() *file {
	return f.bFile
}

func (f *funcDecl) isMethod() bool {
	return f.fType.recv != nil
}

func (f *funcDecl) compEval() bool {
	return f.fType.compEval
}

func (f *funcDecl) source() ast.Node {
	return f.src
}

func (f *funcDecl) resolveOrder() int {
	return 0
}

func (f *funcDecl) buildSignature(fScope *fileResolveScope) (ir.Func, fnResolveScope, bool) {
	ext := &ir.FuncDecl{Src: f.src, FFile: fScope.irFile()}
	var fnscope *funcResolveScope
	var ok bool
	ext.FType, fnscope, ok = f.fType.buildFuncType(fScope)
	return ext, fnscope, ok && fnscope.setFuncValue(ext)
}

func (f *funcDecl) buildScopeBody(rscope resolveScope, ftype *ir.FuncType) (*funcResolveScope, bool) {
	ce, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return nil, false
	}
	ftype, ok := instantiateFType(ce, ftype.Expr(), rscope.fileScope().irFile(), ftype)
	if !ok {
		return nil, false
	}
	return newFuncScope(rscope, ftype)
}

func (f *funcDecl) buildBody(fnscope fnResolveScope, extF *irFunc) bool {
	// Rebuilding the function scope to make sure all function declarations are included.
	fScope, ok := fnscope.fileScope().pkgResolveScope.fileScope(f.bFile)
	if !ok {
		return false
	}
	fnScope, ok := f.buildScopeBody(fScope, extF.irFunc.FuncType())
	if !ok {
		return false
	}
	ext := extF.irFunc.(*ir.FuncDecl)
	ext.Body, ok = buildFuncBody(fnScope, f.body)
	return ok
}

func (f *funcDecl) fnSource() *ast.FuncDecl {
	return f.src
}

func (f *funcDecl) buildAnnotations(*fileResolveScope, *irFunc) bool {
	return true
}

// String returns a string representation of the function.
func (f *funcDecl) String() string {
	return fnName(f)
}

// funcBuiltin is a function imported from a package.
type funcBuiltin struct {
	*funcDecl
}

var _ function = (*funcBuiltin)(nil)

func (bFile *file) processBuiltinFunc(scope procScope, src *ast.FuncDecl, compEval bool) (function, bool) {
	fDecl, declOk := newFuncDecl(scope, src, compEval)
	fn := &funcBuiltin{funcDecl: fDecl}
	return fn, declOk
}

func (f *funcBuiltin) buildSignature(fScope *fileResolveScope) (ir.Func, fnResolveScope, bool) {
	ext := &ir.FuncBuiltin{Src: f.src, FFile: fScope.irFile()}
	var fnScope *funcResolveScope
	var ok bool
	ext.FType, fnScope, ok = f.fType.buildFuncType(fScope)
	if !ok {
		return ext, nil, false
	}
	setOk := fnScope.setFuncValue(ext)
	return ext, fnScope, ok && setOk
}

func (f *funcBuiltin) buildBody(fnResolveScope, *irFunc) bool {
	return true
}

type funcLiteral struct {
	src   *ast.FuncLit
	file  *file
	ftype *funcType
	body  *blockStmt
}

var _ exprNode = (*funcLiteral)(nil)

func processFuncLit(pscope procScope, src *ast.FuncLit) (*funcLiteral, bool) {
	ftype, ftypeOk := processFuncType(pscope, src.Type, nil, false)
	body, bodyOk := processBlockStmt(pscope, src.Body)
	return &funcLiteral{
		src:   src,
		file:  pscope.file(),
		ftype: ftype,
		body:  body,
	}, ftypeOk && bodyOk
}

func (fn *funcLiteral) buildFuncLit(rscope resolveScope) (*ir.FuncLit, bool) {
	lit := &ir.FuncLit{
		Src:   fn.src,
		FFile: rscope.fileScope().irFile(),
	}
	var ok bool
	var fnscope *funcResolveScope
	lit.FType, fnscope, ok = fn.ftype.buildFuncType(rscope)
	if !ok {
		return lit, false
	}
	lit.Body, ok = buildFuncBody(fnscope, fn.body)
	return lit, ok
}

func (fn *funcLiteral) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	lit, ok := fn.buildFuncLit(rscope)
	if !ok {
		return invalidExpr(), false
	}
	return ir.NewFuncValExpr(lit, lit), true
}

func (fn *funcLiteral) source() ast.Node {
	return fn.src
}

func (fn *funcLiteral) String() string {
	return fmt.Sprintf("%s{...}", fn.ftype.String())
}

func funcDeclarator(fn ir.PkgFunc) irb.Declarator {
	return func(decls *ir.Declarations) {
		decls.Funcs = append(decls.Funcs, fn)
	}
}
