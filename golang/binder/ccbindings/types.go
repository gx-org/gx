package ccbindings

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
)

func (b *binder) ccTypeFromKind(knd ir.Kind) (string, error) {
	switch knd {
	case ir.BoolKind:
		return "bool", nil
	case ir.Float32Kind:
		return "float", nil
	case ir.Float64Kind:
		return "double", nil
	case ir.Int32Kind:
		return "int32_t", nil
	case ir.Int64Kind:
		return "int64_t", nil
	case ir.Uint32Kind:
		return "uint32_t", nil
	case ir.Uint64Kind:
		return "uint64_t", nil
	default:
		return "", errors.Errorf("cannot convert kind %s to a C++ type: not supported", knd.String())
	}
}

func (b *binder) ccTypeFromIR(typ ir.Type) (string, error) {
	arrayType, ok := typ.(ir.ArrayType)
	if !ok {
		return "", errors.Errorf("cannot convert %T to a C++ type: not supported", typ)
	}
	if arrayType.Rank().IsAtomic() {
		kind, err := b.ccTypeFromKind(arrayType.Kind())
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("::gxlang::cppgx::Atomic<%s>", kind), nil
	}
	kind, err := b.ccTypeFromKind(arrayType.DataType().Kind())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("::gxlang::cppgx::Array<%s>", kind), nil
}

func (b *binder) ccReturnTypeFromIR(typ ir.Type) (string, error) {
	typS, err := b.ccTypeFromIR(typ)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("absl::StatusOr<%s>", typS), nil
}
