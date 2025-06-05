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

package gobindings

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/binder/bindings"
)

type pkgTypes struct {
	*binder
	ir.StructType
}

func (b *binder) packagePrefixNameOf(tp *ir.NamedType) string {
	pkg := tp.Package()
	if pkg == b.Package {
		return ""
	}
	return b.namePackage(pkg) + "."
}

func (b *binder) nameSlice(tp *ir.SliceType) (string, error) {
	hostType, err := b.bridgerType(tp.DType.Typ)
	if err != nil {
		return "", err
	}
	return strings.Repeat("[]", tp.Rank) + hostType, nil
}

func (b *binder) bridgerType(tp ir.Type) (string, error) {
	switch typT := tp.(type) {
	case ir.ArrayType:
		if typT.Rank().IsAtomic() {
			goType, err := b.nameGoType(typT)
			return fmt.Sprintf("types.Atom[%s]", goType), err
		}
		goType, err := b.nameGoType(typT.DataType())
		return fmt.Sprintf("types.Array[%s]", goType), err
	case *ir.SliceType:
		dtype, err := b.bridgerType(typT.DType.Typ)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("*types.Slice[%s]", dtype), nil
	case *ir.NamedType:
		return "*" + b.packagePrefixNameOf(typT) + fmt.Sprintf("%s", typT.Name()), nil
	default:
		return "", errors.Errorf("cannot write nameBackendType bindings for type %T on device", typT)
	}
}

func (b *binder) gxValueTypePointer(tp ir.Type) (string, error) {
	s, err := b.gxValueType(tp)
	if err != nil {
		return "", err
	}
	switch tp.(type) {
	case *ir.NamedType:
		return "*" + s, nil
	default:
		return s, nil
	}
}

func (b *binder) gxValueType(tp ir.Type) (string, error) {
	switch typT := tp.(type) {
	case ir.ArrayType:
		return "values.Array", nil
	case *ir.NamedType:
		return b.packagePrefixNameOf(typT) + typT.Name(), nil
	case *ir.SliceType:
		return "*values.Slice", nil
	default:
		return "", errors.Errorf("cannot write nameDeviceType bindings for type %T on device", typT)
	}
}

func (b *binder) nameGoType(tp ir.Type) (string, error) {
	switch typT := tp.(type) {
	case ir.ArrayType:
		if !typT.Rank().IsAtomic() {
			return "", errors.Errorf("no Go type name for %T", typT)
		}
		kind := typT.Kind()
		if bindings.IsDefaultInt(kind) {
			return "ir.Int", nil
		}
		return kind.String(), nil
	case *ir.NamedType:
		return b.nameGoType(typT.Underlying.Typ)
	default:
		return "", errors.Errorf("cannot write nameGoType bindings for type %T on device", typT)
	}
}

func (b *binder) nameHostFuncType(tp ir.Type) (string, error) {
	switch typT := tp.(type) {
	case ir.ArrayType:
		if !typT.Rank().IsAtomic() {
			return "", errors.Errorf("no host function for %T", typT)
		}
		return toKindSuffix(typT), nil
	case *ir.NamedType:
		return b.nameHostFuncType(typT.Underlying.Typ)
	default:
		return "", errors.Errorf("cannot write nameHostFuncType bindings for type %T on device", typT)
	}
}
