package cpevelements

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

type typeParam struct {
	src elements.ExprAt
	typ *ir.TypeParam
}

func newTypeParam(src elements.ExprAt, typ *ir.TypeParam) elements.Element {
	return &typeParam{src: src, typ: typ}
}

func (tp *typeParam) Flatten() ([]elements.Element, error) {
	return []elements.Element{tp}, nil
}

// Unflatten creates a GX value from the next handles available in the Unflattener.
func (tp *typeParam) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	return nil, errors.Errorf("not implemented")
}

// Kind returns the kind of the element being stored.
func (tp *typeParam) Kind() ir.Kind {
	return tp.typ.Kind()
}
