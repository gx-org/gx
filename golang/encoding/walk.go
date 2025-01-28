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

// Package encoding encodes GX structure
package encoding

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
)

type (
	// DataSlice is serialized data of a slice.
	DataSlice interface {
		Index(int) (Data, error)
		Len() int
	}

	// DataStruct is serialized data of a struct.
	DataStruct interface {
		Field(name string) (Data, error)

		TagName() string
	}

	// BridgeProcessor processes a bridge instance once it has been loaded.
	BridgeProcessor func(types.ArrayBridge) (types.ArrayBridge, error)

	// ValueFuture is a proxy to a concrete value being loaded
	// perhaps in the background.
	ValueFuture interface {
		Value() (types.ArrayBridge, error)
	}

	// Data is a generic kind of data.
	Data interface {
		// ToDataSlice returns a slice view on the serialized data.
		ToDataSlice() (DataSlice, error)

		// ToDataStruct returns a struct view on the serialized data.
		ToDataStruct() (DataStruct, error)

		// ValueFuture able to return data once it has been loaded.
		ValueFuture() (ValueFuture, error)
	}
)

// SendToDevice returns a bridge processor to send arrays to the device when they are loaded.
func SendToDevice(dev *api.Device) BridgeProcessor {
	return func(arr types.ArrayBridge) (types.ArrayBridge, error) {
		return arr.ToDevice(dev)
	}

}

func unmarshal(ld *loader, target types.Bridge, data Data) error {
	var err error
	kind := target.GXValue().Type().Kind()
	switch kind {
	case ir.StructKind:
		err = unmarshalStruct(ld, target, data)
	case ir.SliceKind:
		err = unmarshalSliceType(ld, target, data)
	default:
		err = errors.Errorf("type %T not supported", kind)
	}
	return err
}

func unmarshalStruct(ld *loader, target types.Bridge, data Data) error {
	dataStruct, err := data.ToDataStruct()
	if err != nil {
		return err
	}
	structTarget, ok := target.(types.StructBridge)
	if !ok {
		return errors.Errorf("target %T does not implement %s", target, reflect.TypeFor[types.StructBridge]().String())
	}
	structType := structTarget.StructValue().StructType()
	for _, field := range structType.Fields.Fields() {
		fieldName := field.Name.Name
		if !ir.IsExported(fieldName) {
			continue
		}
		key := keyFromField(dataStruct.TagName(), field)
		fieldData, err := dataStruct.Field(key)
		if err != nil {
			return err
		}
		if IsValue(field.Type()) {
			if err := ld.setNumerical(fieldData, func(val types.Bridge) error {
				return structTarget.SetField(field, val)
			}); err != nil {
				return err
			}
			continue
		}
		fieldBridge, err := structTarget.NewFromField(field)
		if err != nil {
			return err
		}
		if err = unmarshal(ld, fieldBridge, fieldData); err != nil {
			return err
		}
		if err := structTarget.SetField(field, fieldBridge); err != nil {
			return fmt.Errorf("cannot set field %s: %w", fieldName, err)
		}
	}
	return nil
}

func unmarshalSliceElement(ld *loader, i int, sliceTarget *types.SliceBridge, sliceData DataSlice) error {
	elementData, err := sliceData.Index(i)
	if err != nil {
		return err
	}
	if IsValue(sliceTarget.DType()) {
		return ld.setNumerical(elementData, func(val types.Bridge) error {
			return sliceTarget.Set(i, val)
		})
	}
	elementTarget := sliceTarget.Get(i)
	if elementTarget == nil {
		return errors.Errorf("element %d of the slice has not been allocated", i)
	}
	if err := unmarshal(ld, elementTarget, elementData); err != nil {
		return errors.Errorf("cannot set element %d of the slice: %v", i, err)
	}
	return nil
}

func unmarshalSliceType(ld *loader, target types.Bridge, data Data) error {
	sliceData, err := data.ToDataSlice()
	if err != nil {
		return err
	}
	sliceTarget, ok := target.(*types.SliceBridge)
	if !ok {
		return errors.Errorf("cannot cast %T to %s", target, reflect.TypeFor[*types.SliceBridge]().String())
	}
	sliceTarget.Allocate(sliceData.Len())
	for i := 0; i < sliceData.Len(); i++ {
		if err := unmarshalSliceElement(ld, i, sliceTarget, sliceData); err != nil {
			return errors.Errorf("cannot set element %d of the slice: %v", i, err)
		}
	}
	return nil
}

func keyFromField(key string, field *ir.Field) string {
	name := field.Name.Name
	tag := field.Group.Src.Tag
	if tag == nil {
		return name
	}
	tagS := tag.Value
	tagS = strings.TrimLeft(tagS, "`")
	tagS = strings.TrimRight(tagS, "`")
	if tagName, ok := reflect.StructTag(tagS).Lookup(key); ok {
		return tagName
	}
	return name
}

// Unmarshal populates a GX structure from a Go map.
func Unmarshal(bProc BridgeProcessor, target types.Bridger, data Data) (err error) {
	loader := newLoader(bProc)
	defer func() {
		errClose := loader.close()
		if err == nil {
			// We only report closing error if we had no previous errors.
			err = errClose
		}
	}()
	err = unmarshal(loader, target.Bridge(), data)
	return
}

// IsValue returns true if the field is a value that can be set, as opposed
// to a composite structure that needs to be filled recursively.
func IsValue(typ ir.Type) bool {
	_, ok := typ.(ir.ArrayType)
	return ok
}
