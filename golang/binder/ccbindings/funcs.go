package ccbindings

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
)

type (
	funcField struct {
		Name      string
		Type      string
		Separator string
	}

	function struct {
		*binder
		ir.Func
		FuncIndex   int
		ReturnType  string
		Params      []funcField
		Results     []funcField
		ReturnTuple bool
	}
)

func (b *binder) newFunc(f ir.Func, i int) (*function, error) {
	fn := &function{
		binder:      b,
		Func:        f,
		FuncIndex:   i,
		ReturnTuple: f.FuncType().Results.Len() > 1,
	}
	var err error
	if fn.ReturnType, err = fn.returnType(); err != nil {
		return nil, err
	}
	if fn.Params, err = fn.processFields(f.FuncType().Params, "param"); err != nil {
		return nil, err
	}
	if fn.Results, err = fn.processFields(f.FuncType().Results, "result"); err != nil {
		return nil, err
	}
	return fn, nil
}

func (f function) processFields(fieldList *ir.FieldList, prefix string) ([]funcField, error) {
	var fFields []funcField
	fields := fieldList.Fields()
	for i, field := range fields {
		typ, err := f.binder.ccTypeFromIR(field.Group.Type)
		if err != nil {
			return nil, err
		}
		name := field.Name.Name
		if name == "" {
			name = fmt.Sprintf("%s%02d", prefix, i)
		}
		param := funcField{
			Name: name,
			Type: typ,
		}
		if i < len(fields)-1 {
			param.Separator = ", "
		}
		fFields = append(fFields, param)
	}
	return fFields, nil
}

func (f function) returnType() (string, error) {
	res := f.FuncType().Results
	if res.Len() == 0 {
		return "void", nil
	}
	if res.Len() == 1 {
		return f.binder.ccReturnTypeFromIR(res.List[0].Type)
	}
	return "", errors.Errorf("returning more than one result not supported for C++ bindings")
}
