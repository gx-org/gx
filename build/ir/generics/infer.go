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
		defined map[string]ir.Type
		axes    map[string]ir.Element
	}

	argUnifier struct {
		*unifier
		arg ir.AssignableExpr
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

func (uni *argUnifier) Source() ast.Node {
	return uni.arg.Source()
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
	ok, err := ir.AssignableTo(uni, typ, tp.Field.Type())
	if err != nil {
		uni.defined[name] = ir.InvalidType()
		return uni.Err().AppendAt(uni.arg.Source(), err)
	}
	if !ok {
		uni.defined[name] = ir.InvalidType()
		return uni.Err().Appendf(uni.arg.Source(), "%s does not satisfy %s for %s", typ.String(), tp.Field.Type().String(), tp.Field.Name.Name)
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
	eq, err := defined.Equal(uni, typ)
	if err != nil {
		return uni.Err().AppendAt(uni.arg.Source(), err)
	}
	if !eq {
		return uni.Err().Appendf(uni.arg.Source(), "type %s does not match type %s for %s", typ.String(), defined.String(), tp.Field.Name.Name)
	}
	return true
}

func (uni *argUnifier) defineAxis(param *ir.AxisStmt, targets []ir.AxisLengths) bool {
	if len(targets) == 0 {
		return uni.Err().Appendf(uni.arg.Source(), "no axis left to define %s", param.NameDef().Name)
	}
	ax := targets[0]
	el, err := uni.Fetcher.EvalExpr(ax.AsExpr())
	if err != nil {
		return uni.Err().AppendAt(uni.Source(), err)
	}
	return uni.defineAxisElement(param, el)
}

func (uni *argUnifier) defineAxisElement(param *ir.AxisStmt, el ir.Element) bool {
	name := param.NameDef().Name
	defined, isDefined := uni.axes[name]
	if !isDefined {
		uni.axes[name] = el
		return true
	}
	eq, err := ir.ElementEqual(defined, el)
	if err != nil {
		return uni.Err().AppendAt(uni.arg.Source(), err)
	}
	if !eq {
		return uni.Err().Appendf(uni.arg.Source(), "axis length %v does not match length %v for %s", defined, el, name)
	}
	return true
}

func (uni *argUnifier) defineGroupAsAllSingleAxes(param *ir.AxisStmt, targets []ir.AxisLengths) ([]ir.AxisLengths, bool) {
	var singles []ir.Element
	for _, axis := range targets {
		if axis.Type().Kind() != irkind.IntLen {
			break
		}
		el, err := uni.Fetcher.EvalExpr(axis.AsExpr())
		if err != nil {
			return nil, uni.Fetcher.Err().AppendAt(uni.Source(), err)
		}
		singles = append(singles, el)
	}
	return targets[len(singles):], uni.defineAxisElement(param, elements.NewSlice(
		ir.IntLenSliceType(), singles,
	))
}

func (uni *argUnifier) defineGroupAxis(param *ir.AxisStmt, targets []ir.AxisLengths) ([]ir.AxisLengths, bool) {
	if len(targets) == 0 {
		return uni.defineGroupAsAllSingleAxes(param, targets)
	}
	switch targets[0].Type().Kind() {
	case irkind.IntLen:
		return uni.defineGroupAsAllSingleAxes(param, targets)
	case irkind.Slice:
		ok := uni.defineAxis(param, targets)
		return targets[1:], ok
	default:
		arg := targets[0]
		return nil, uni.Err().Appendf(uni.Source(), "cannot unify axis length %s of type %s in parameters with axis length %s of type %s: not supported", param.String(), param.Type().String(), arg.String(), arg.Type().String())
	}
}

func (uni *argUnifier) DefineAxis(param *ir.AxisStmt, targets []ir.AxisLengths) ([]ir.AxisLengths, bool) {
	switch param.Type().Kind() {
	case irkind.IntLen:
		ok := uni.defineAxis(param, targets)
		return targets[1:], ok
	case irkind.Slice:
		return uni.defineGroupAxis(param, targets)
	default:
		return nil, uni.Err().Appendf(uni.Source(), "cannot unify axis expression of type %s in parameters: not supported", param.Type().String())
	}
}

// Infer the type parameters of a function given a list of argument expressions.
func Infer(fetcher ir.Fetcher, fExpr *ir.FuncValExpr, args []ir.AssignableExpr) (*ir.FuncValExpr, bool) {
	ftype := fExpr.T
	uni := &unifier{
		Fetcher: fetcher,
		defined: newTypeParamDefinition(fExpr.T),
		axes:    make(map[string]ir.Element),
	}
	ok := true
	for i, param := range ftype.Params.Fields() {
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
	subFetcher, ok := fetcher.Sub(uni.axes)
	if !ok {
		return fExpr, false
	}
	spec := &specialiser{
		Fetcher: subFetcher,
		defined: uni.defined,
		axes:    uni.axes,
	}
	ftypeInfer, err := ftype.SpecialiseFType(spec)
	if err != nil {
		return fExpr, fetcher.Err().AppendAt(fExpr.X.Source(), err)
	}
	return &ir.FuncValExpr{
		X: fExpr.X,
		F: fExpr.F,
		T: ftypeInfer,
	}, ok
}
