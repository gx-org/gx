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

package ccbindings

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

func (b *binder) ccTypeFromKind(knd irkind.Kind) (string, error) {
	switch knd {
	case irkind.Bool:
		return "bool", nil
	case irkind.Float32:
		return "float", nil
	case irkind.Float64:
		return "double", nil
	case irkind.Int32:
		return "int32_t", nil
	case irkind.Int64:
		return "int64_t", nil
	case irkind.Uint32:
		return "uint32_t", nil
	case irkind.Uint64:
		return "uint64_t", nil
	case irkind.IntLen:
		return "uint64_t", nil
	default:
		return "", errors.Errorf("cannot convert kind %s to a C++ type: not supported", knd.String())
	}
}

func (b *binder) ccTypeFromIR(typ ir.Type) (string, error) {
	if tpl, ok := typ.(*ir.TupleType); ok {
		var typeNames []string
		for _, typ := range tpl.Types {
			typeName, err := b.ccTypeFromIR(typ)
			if err != nil {
				return "", err
			}
			typeNames = append(typeNames, typeName)
		}
		return fmt.Sprintf("std::tuple<%s>", strings.Join(typeNames, ", ")), nil
	}

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
	if typ.Kind() == irkind.Void {
		return "absl::Status", nil
	}
	typS, err := b.ccTypeFromIR(typ)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("absl::StatusOr<%s>", typS), nil
}
