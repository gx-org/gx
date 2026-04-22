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

package generics

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/interp/elements"
)

type (
	unifier struct {
		ir.Fetcher
		params  map[string]*ir.Field
		defined map[string]ir.Type
		axes    map[string]ir.Element
	}

	argUnifier struct {
		*unifier
		arg ir.Expr
	}
)

var _ = (ir.Unifier)(&argUnifier{})

func (uni *unifier) specialiseRemainingNumbers() bool {
	for name, v := range uni.defined {
		if !irkind.IsNumber(v.Kind()) {
			continue
		}
		uni.defined[name] = ir.DefaultNumberType(v.Kind())
	}
	return true
}

func (uni *argUnifier) Node() ast.Node {
	return uni.arg.Node()
}

func (uni *argUnifier) specialiseNumber(name string, defined, typ ir.Type) ir.Type {
	if !irkind.IsNumber(defined.Kind()) {
		return defined
	}
	if irkind.IsNumber(typ.Kind()) {
		return defined
	}
	if !ir.CanBeNumber(typ) {
		return defined
	}
	uni.defined[name] = typ
	return typ
}

func (uni *argUnifier) DefineTParam(tp *ir.TypeParam, typ ir.Type) bool {
	name := tp.Field.Name.Name
	ok, cpErr, err := ir.AssignableTo(uni, typ, tp.Field.Type())
	if err != nil {
		uni.defined[name] = ir.InvalidType()
		return uni.Err().AppendAt(uni.arg.Node(), err)
	}
	if cpErr != nil {
		uni.defined[name] = ir.InvalidType()
		return uni.Err().AppendAt(uni.arg.Node(), cpErr)
	}
	if !ok {
		uni.defined[name] = ir.InvalidType()
		return uni.Err().Appendf(uni.arg.Node(), "%s does not satisfy %s for %s", typ.ReferString(uni.File()), tp.Field.Type().ReferString(uni.File()), tp.Field.Name.Name)
	}
	defined := uni.defined[name]
	if defined == nil {
		uni.defined[name] = typ
		return true
	}
	defined = uni.specialiseNumber(name, defined, typ)
	if !ir.IsValid(defined) || !ir.IsValid(typ) {
		return false
	}
	eq, cpErr, err := defined.Equal(uni, typ)
	if err != nil {
		return uni.Err().AppendAt(uni.arg.Node(), err)
	}
	if cpErr != nil {
		return uni.Err().AppendAt(uni.arg.Node(), cpErr)
	}
	if !eq {
		return uni.Err().Appendf(uni.arg.Node(), "type %s does not match type %s for %s", typ.ReferString(uni.File()), defined.ReferString(uni.File()), tp.Field.Name.Name)
	}
	return true
}

func (uni *argUnifier) defineAxis(name string, targets []ir.AxisLengths) bool {
	if len(targets) == 0 {
		return uni.Err().Appendf(uni.arg.Node(), "no axis left to define %s", name)
	}
	ax := targets[0]
	el, err := uni.Fetcher.EvalExpr(ax.AsExpr())
	if err != nil {
		return uni.Err().AppendAt(uni.Node(), err)
	}
	return uni.defineAxisElement(name, el)
}

func (uni *argUnifier) defineAxisElement(name string, el ir.Element) bool {
	defined, isDefined := uni.axes[name]
	if !isDefined {
		uni.axes[name] = el
		return true
	}
	eq, err := ir.ElementEqual(defined, el)
	if err != nil {
		return uni.Err().AppendAt(uni.arg.Node(), err)
	}
	if !eq {
		return uni.Err().Appendf(uni.arg.Node(), "axis length %v does not match length %v for %s", defined, el, name)
	}
	return true
}

func (uni *argUnifier) defineGroupAsAllSingleAxes(src ast.Node, name string, targets []ir.AxisLengths) ([]ir.AxisLengths, bool) {
	var singles []ir.Element
	for _, axis := range targets {
		if axis.Type().Kind() != irkind.IntLen {
			break
		}
		el, err := uni.Fetcher.EvalExpr(axis.AsExpr())
		if err != nil {
			return nil, uni.Fetcher.Err().AppendAt(uni.Node(), err)
		}
		singles = append(singles, el)
	}
	slice, err := elements.NewSlice(
		ir.IntLenSliceType(), singles,
	)
	sliceOk := true
	if err != nil {
		sliceOk = uni.Err().AppendAt(src, err)
	}
	return targets[len(singles):], uni.defineAxisElement(name, slice) && sliceOk
}

func (uni *argUnifier) defineGroupAxis(src ast.Node, name string, tp ir.Type, targets []ir.AxisLengths) ([]ir.AxisLengths, bool) {
	if len(targets) == 0 {
		return uni.defineGroupAsAllSingleAxes(src, name, targets)
	}
	switch targets[0].Type().Kind() {
	case irkind.IntLen:
		return uni.defineGroupAsAllSingleAxes(src, name, targets)
	case irkind.Slice:
		ok := uni.defineAxis(name, targets)
		return targets[1:], ok
	default:
		arg := targets[0]
		return nil, uni.Err().Appendf(uni.Node(), "cannot unify axis length %s of type %s in parameters with axis length %s of type %s: not supported", name, tp.ReferString(uni.File()), arg.SourceString(uni.File()), arg.Type().ReferString(uni.File()))
	}
}

func (uni *argUnifier) DefineAxis(src ast.Node, name string, tp ir.Type, targets []ir.AxisLengths) ([]ir.AxisLengths, bool) {
	switch tp.Kind() {
	case irkind.IntLen:
		ok := uni.defineAxis(name, targets)
		if len(targets) < 1 {
			return nil, true
		}
		return targets[1:], ok
	case irkind.Slice:
		return uni.defineGroupAxis(src, name, tp, targets)
	default:
		return nil, uni.Err().Appendf(uni.Node(), "cannot unify axis expression of type %s in parameters: not supported", tp.ReferString(uni.File()))
	}
}

func (uni *argUnifier) IsTypeParam(id *ir.Ident) bool {
	if !ir.ValidIdent(id.Src) {
		return false
	}
	return uni.params[id.Src.Name] != nil
}

func typeParametersMap(tparams *ir.FieldList) map[string]*ir.Field {
	fields := make(map[string]*ir.Field)
	for _, field := range tparams.Fields() {
		if !ir.ValidIdent(field.Name) {
			continue
		}
		fields[field.Name.Name] = field
	}
	return fields
}

// Infer the type parameters of a function given a list of argument expressions.
func Infer(fetcher ir.Fetcher, fExpr *ir.FuncValExpr, args []ir.Expr) (*ir.FuncValExpr, bool) {
	ftype := fExpr.FuncType()
	uni := &unifier{
		Fetcher: fetcher,
		defined: newTypeParamDefinition(ftype),
		axes:    make(map[string]ir.Element),
		params:  typeParametersMap(fExpr.FuncType().TypeParams),
	}
	ok := true
	for i, param := range ftype.Params.Fields() {
		if i >= len(args) {
			// We may have less argument than parameters.
			// For example:
			//   func f(...int32)
			// called with:
			//   f()
			break
		}
		argUni := &argUnifier{unifier: uni, arg: args[i]}
		if argOk := param.Type().UnifyWith(argUni, argUni.arg.Type()); !argOk {
			ok = false
		}
	}
	if !ok {
		return fExpr, false
	}
	if !uni.specialiseRemainingNumbers() {
		return fExpr, false
	}
	subFetcher, err := fetcher.Sub(nil, uni.axes)
	if err != nil {
		return fExpr, fetcher.Err().AppendAt(fExpr.Node(), err)
	}
	spec := newSpecialiser(subFetcher, fExpr.Func(), uni.defined, uni.axes)
	ftypeInfer, cpErr, err := ftype.SpecialiseFType(spec)
	if cpErr != nil {
		return fExpr, fetcher.Err().AppendAt(fExpr.Node(), cpErr)
	}
	if err != nil {
		return fExpr, fetcher.Err().AppendAt(fExpr.Node(), err)
	}
	return fExpr.NewFType(ftypeInfer), ok
}
