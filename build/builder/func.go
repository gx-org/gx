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
	"math/big"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/base/ordered"
	"github.com/gx-org/gx/build/builder/irb"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir/generics"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
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
	typ := &ir.TypeParam{Field: fieldStorage.Field}
	// Transform storage with a type to a storage with the type being assigned as a value.
	return defineLocalVar(s, &ir.AssignExpr{
		Storage: &ir.LocalVarStorage{
			Src: storage.NameDef(),
			Typ: ir.MetaType(),
		},
		X: ir.TypeExpr(nil, typ),
	})
}

type ftypeAxisLengths struct {
	axLens []ir.AxisValue
}

func (axs *ftypeAxisLengths) define(s resolveScope, storage ir.Storage) bool {
	axStmt, ok := storage.(*ir.AxisStmt)
	if !ok {
		return s.Err().AppendInternalf(storage.NameDef(), "cannot register axis %s: cannot cast %T to %s", storage.NameDef().Name, storage, reflect.TypeFor[*ir.AxisStmt]())
	}
	axs.axLens = append(axs.axLens, ir.AxisValue{Axis: axStmt})
	return defineLocalVar(s, storage)
}

func (n *funcType) buildFuncType(rscope resolveScope) (*ir.FuncType, *funcResolveScope, bool) {
	ext := &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: n.src},
		CompEval: n.compEval,
	}
	var tParamsOk, recvOk, paramsOk, resultsOk bool
	sigscope, ok := newEphemeralResolveScope(rscope, n.src)
	if !ok {
		return ext, nil, false
	}
	typeParamsScope := newDefineScope(sigscope, defineTypeParam, nil)
	ext.TypeParams, tParamsOk = n.typeParams.buildFieldList(typeParamsScope)
	ext.Receiver, recvOk = n.recv.buildFieldList(newDefineScope(sigscope, nil, nil))
	if recvOk && ext.Receiver != nil {
		if field := ext.ReceiverField(); field.Name != nil {
			defineLocalVar(sigscope, field.Storage())
		}
	}
	axisLengths := &ftypeAxisLengths{}
	paramScope := newDefineScope(sigscope, defineLocalVar, axisLengths.define)
	ext.Params, paramsOk = n.params.buildFieldList(paramScope)
	if !paramsOk {
		return ext, nil, false
	}
	resultScope := newDefineScope(sigscope, defineLocalVar, nil)
	ext.Results, resultsOk = n.results.buildFieldList(resultScope)
	if resultsOk {
		for _, field := range ext.Results.Fields() {
			if !rankInferOk(rscope, field.Type().Node(), field.Type()) {
				resultsOk = false
			}
		}
	}
	ext.AxisLengths = axisLengths.axLens
	fnscope, fnscopeOk := newFuncScope(rscope, ext)
	return ext, fnscope, tParamsOk && paramsOk && resultsOk && recvOk && fnscopeOk
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
	case annotator:
		fn, ok = bFile.processAnnotatorFunc(fileScope, src, dirComment)
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
	var ok bool
	var fnscope *funcResolveScope
	ext.FType, fnscope, ok = f.fType.buildFuncType(fScope)
	if !ok {
		return ext, nil, false
	}
	fnscope, ok = fnscope.setFuncValue(ext)
	return ext, fnscope, ok
}

func (f *funcDecl) buildBody(fnscope fnResolveScope, extF *irFunc) bool {
	// Rebuilding the function scope to make sure all function declarations are included.
	fScope, ok := fnscope.fileScope().pkgResolveScope.newFileRScope(f.bFile)
	if !ok {
		return false
	}
	fnScope, ok := newFuncScope(fScope, extF.irFunc.FuncType())
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
	var ok bool
	var fnScope *funcResolveScope
	ext.FType, fnScope, ok = f.fType.buildFuncType(fScope)
	if !ok {
		return ext, nil, false
	}
	fnScope, ok = fnScope.setFuncValue(ext)
	return ext, fnScope, ok
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
	return fmt.Sprintf("func %s{...}", fn.ftype.String())
}

func convertArgNumbers(rscope resolveScope, fType *ir.FuncType, args []ir.Expr) ([]ir.Expr, bool) {
	args = append([]ir.Expr{}, args...)
	params := fType.Params.Fields()
	argsOk := true
	for i, arg := range args {
		if !irkind.IsNumber(arg.Type().Kind()) {
			continue
		}
		var iOk bool
		args[i], iOk = castNumber(rscope, arg, params[i].Type())
		argsOk = argsOk && iOk
	}
	return args, argsOk
}

func axisExprFrom(rscope resolveScope, ax ir.AxisLengths) (*ir.AxisExpr, bool) {
	if ax == nil {
		return nil, rscope.Err().Append(fmterr.Internal(errors.Errorf("axis length is nil")))
	}
	switch axisT := ax.(type) {
	case *ir.AxisExpr:
		return axisT, true
	case *ir.AxisInfer:
		return axisExprFrom(rscope, axisT.X)
	case *ir.AxisStmt:
		return &ir.AxisExpr{X: axisT.AsExpr()}, true
	}
	return nil, rscope.Err().AppendInternalf(ax.Node(), "unknown axis length type: %T:%s", ax, ax.String())
}

func axisValuesFromArgumentValue(rscope resolveScope, compEval *compileEvaluator, src *ir.Field, val ir.Element) ([]ir.Element, bool) {
	arrayElement, axOk := val.(elements.WithAxes)
	if !axOk {
		return nil, true
	}
	axes, err := arrayElement.Axes(compEval)
	if err != nil {
		return nil, rscope.Err().AppendInternalf(src.Node(), "cannot get axes from element %T to assign to parameter %s: %v", val, src.Name, err)
	}
	if axes == nil {
		return nil, true
	}
	return axes.Elements(), true
}

var (
	cstFile = &ir.File{
		Package: &ir.Package{
			Name: &ast.Ident{Name: "__gx_builder_package"},
		},
		Src: &ast.File{Name: &ast.Ident{Name: "__gx_builder_file"}},
	}
	zeroExpr = &ir.NumberCastExpr{
		X: &ir.NumberInt{
			Val: &big.Int{},
		},
		Typ: ir.IntLenType(),
	}
	zeroValue, _ = values.AtomNumberInt(&big.Int{}, zeroExpr.Type())
	zeroLen, _   = cpevelements.NewAtom(elements.NewExprAt(cstFile, zeroExpr), zeroValue)
	emptySlice   = elements.NewSlice(ir.IntLenSliceType(), nil)
)

func buildAtomicAxisValue(rscope resolveScope, arg ir.Expr, elts []ir.Element) (ax ir.Element, todo []ir.Element) {
	if len(elts) == 0 {
		return zeroLen, nil
	}
	return elts[0], elts[1:]
}

func buildSliceAxisValue(rscope resolveScope, arg ir.Expr, elts []ir.Element) (ax ir.Element, todo []ir.Element) {
	if len(elts) == 0 {
		return emptySlice, nil
	}
	return elements.NewSlice(arg.Type(), elts), nil
}

func assignArgValueToName(rscope resolveScope, compEval *compileEvaluator, params map[string]ir.Element, param *ir.Field, arg ir.Expr, argVal ir.Element) bool {
	name := param.Name.Name
	if ir.ValidName(name) {
		params[name] = argVal
	}
	paramArrayType, ok := param.Type().(ir.ArrayType)
	if !ok {
		// The parameter type is not an array: nothing is left to assign,
		// we can return.
		return true
	}
	axisValues, ok := axisValuesFromArgumentValue(rscope, compEval, param, argVal)
	if !ok {
		return ok
	}
	for _, axis := range paramArrayType.Rank().Axes() {
		axExpr, axisOk := axisExprFrom(rscope, axis)
		if !axisOk {
			ok = false
			continue
		}
		ident, isIdent := axExpr.X.(*ir.Ident)
		if !isIdent {
			continue
		}
		if _, isAxisStmt := ident.Stor.(*ir.AxisStmt); !isAxisStmt {
			continue
		}
		var buildAxisValue func(resolveScope, ir.Expr, []ir.Element) (ir.Element, []ir.Element)
		if axExpr.Type().Kind() == irkind.IntLen {
			buildAxisValue = buildAtomicAxisValue
		} else {
			buildAxisValue = buildSliceAxisValue
		}
		params[ident.Src.Name], axisValues = buildAxisValue(rscope, arg, axisValues)
	}
	return ok
}

func assignArgValueToParamName(rscope resolveScope, fExpr *ir.FuncValExpr, args []ir.Expr) (map[string]ir.Element, bool) {
	params := make(map[string]ir.Element)
	compEval, ok := rscope.compEval()
	if !ok {
		return params, false
	}
	for i, param := range fExpr.FuncType().Params.Fields() {
		if param.Name == nil {
			continue
		}
		argVal, err := compEval.fitp.EvalExpr(args[i])
		if err != nil {
			return nil, rscope.Err().AppendAt(fExpr.Node(), err)
		}
		if !assignArgValueToName(rscope, compEval, params, param, args[i], argVal) {
			return nil, false
		}
	}
	return params, true
}

func checkArgsForCall(ce *compileEvaluator, fExpr *ir.FuncValExpr, args []ir.Expr) bool {
	ok := true
	wants := fExpr.FuncType().Params.Fields()
	for i, arg := range args {
		param := wants[i]
		assignable, err := ir.AssignableTo(ce, arg.Type(), param.Type())
		if err != nil {
			return ce.Err().AppendAt(arg.Node(), err)
		}
		if !assignable {
			ok = ce.Err().Appendf(arg.Node(), "cannot use type %s as %s in argument to %s", arg.Type().String(), param.Type().String(), fExpr.Func().ShortString())
		}
	}
	return ok
}

func buildFuncForCall(rscope resolveScope, fExpr *ir.FuncValExpr, args []ir.Expr) ([]ir.Expr, *ir.FuncValExpr, bool) {
	compEval, compEvalOk := compEvalForFuncType(rscope, fExpr.Node(), fExpr.FuncType())
	if !compEvalOk {
		return args, fExpr, false
	}
	var ok bool
	fExpr, ok = generics.Infer(compEval, fExpr, args)
	if !ok {
		return args, fExpr, false
	}
	typeParams := fExpr.FuncType().TypeParams.Fields()
	if len(typeParams) > 0 {
		names := make([]string, len(typeParams))
		for i, field := range typeParams {
			names[i] = field.Name.Name
		}
		parameter := "parameter"
		if len(names) > 1 {
			parameter = "parameters"
		}
		return args, fExpr, rscope.Err().Appendf(fExpr.Node(), "cannot infer type %s %s", parameter, strings.Join(names, ","))
	}
	if args, ok = convertArgNumbers(rscope, fExpr.FuncType(), args); !ok {
		return args, fExpr, false
	}
	argsVals, ok := assignArgValueToParamName(rscope, fExpr, args)
	if !ok {
		return args, fExpr, false
	}
	ce, ok := compEval.sub(argsVals)
	if !ok {
		return args, fExpr, false
	}
	fTypeInst, err := generics.Instantiate(ce, fExpr.FuncType())
	if err != nil {
		return args, fExpr, rscope.Err().AppendAt(fExpr.Node(), err)
	}
	fExprInst := fExpr.NewFType(fTypeInst)
	return args, fExprInst, ok && checkArgsForCall(ce, fExprInst, args)
}

func funcDeclarator(fn ir.PkgFunc) irb.Declarator {
	return func(decls *ir.Declarations) {
		decls.Funcs = append(decls.Funcs, fn)
	}
}
