package generics

import (
	"github.com/gx-org/gx/build/ir"
)

// Infer the type parameters of a function given a list of argument expressions.
func Infer(fetcher ir.Fetcher, fExpr *ir.FuncValExpr, args []ir.AssignableExpr) (*ir.FuncValExpr, bool) {
	fType := fExpr.T
	nameToTypeParam := make(map[string]ir.Type)
	for _, typeParam := range fType.TypeParams.Fields() {
		nameToTypeParam[typeParam.Name.Name] = typeParam.Type()
	}
	if len(nameToTypeParam) == 0 {
		// Nothing to infer.
		return fExpr, true
	}
	nameToType := make(map[string]ir.Type)
	inferredOk := true
	for i, param := range fType.Params.Fields() {
		arg := args[i]
		if ir.IsNumber(arg.Type().Kind()) {
			continue
		}
		gType := extractTypeParamName(param.Group)
		if gType == nil {
			continue
		}
		assigned := nameToType[gType.name()]
		if assigned != nil {
			assignedOk, err := assigned.Equal(fetcher, arg.Type())
			if err != nil {
				inferredOk = fetcher.Err().Append(err)
				continue
			}
			if !assignedOk {
				inferredOk = fetcher.Err().Appendf(arg.Source(), "type %s of %s does not match inferred type %s for %s", arg.Type(), arg.String(), assigned.String(), gType)
				continue
			}
		}
		targetType := nameToTypeParam[gType.name()]
		if targetType == nil {
			continue
		}
		canAssign, err := arg.Type().AssignableTo(fetcher, targetType)
		if err != nil {
			inferredOk = fetcher.Err().Append(err)
		}
		if !canAssign {
			inferredOk = fetcher.Err().Appendf(arg.Source(), "%s does not satisfy %s", arg.Type(), targetType)
		}
		nameToType[gType.name()] = arg.Type()
	}
	if !inferredOk {
		return fExpr, false
	}
	if len(nameToType) == 0 {
		return fExpr, true
	}
	specialised, ok := specialise(fetcher, fExpr.T, nameToType)
	return &ir.FuncValExpr{
		X: fExpr.X,
		F: fExpr.F,
		T: specialised,
	}, ok
}
