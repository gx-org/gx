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

// Package togo provides a mechanism to convert GX values to Go values.
package togo

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
)

// FuncInputs are GX values passed to the function call.
type FuncInputs struct {
	// Receiver on which the function call was done.
	// Can be nil.
	Receiver ir.Element

	// Args returns list of arguments passed to the interpreter at call time.
	Args []ir.Element
}

// CompileValue is an element able to convert its value into an equivalent Go value.
type CompileValue interface {
	ir.Element
	// GoValue of the underlying element.
	GoValue() (any, error)
}

// CpValue converts an element to its Go value.
func CpValue(el ir.Element) (any, error) {
	togo, err := cast.To[CompileValue](ir.BareValue(el))
	if err != nil {
		return nil, err
	}
	return togo.GoValue()
}

// CpValueT converts an element to a specific Go value.
func CpValueT[T any](el ir.Element) (valT T, err error) {
	valAny, err := CpValue(el)
	if err != nil {
		return
	}
	return cast.To[T](valAny)
}

// RuntimeValue is an element able to convert its value into an equivalent Go value.
type RuntimeValue interface {
	ir.Element
	// LastValue of the underlying element for the last call to GX.
	LastValue(*FuncInputs) (any, error)
}

// RtValue converts an element to its Go value.
func RtValue(fnin *FuncInputs, el ir.Element) (any, error) {
	togo, err := cast.To[RuntimeValue](ir.BareValue(el))
	if err != nil {
		return nil, err
	}
	return togo.LastValue(fnin)
}

// RtValueT converts an element to a specific Go value.
func RtValueT[T any](fnin *FuncInputs, el ir.Element) (valT T, err error) {
	valAny, err := RtValue(fnin, el)
	if err != nil {
		return
	}
	return cast.To[T](valAny)
}

// Value converts an element to its Go value.
func Value(fnin *FuncInputs, el ir.Element) (any, error) {
	switch elT := el.(type) {
	case CompileValue:
		return elT.GoValue()
	case RuntimeValue:
		if fnin == nil {
			return nil, errors.Errorf("no input set from the function call")
		}
		return elT.LastValue(fnin)
	}
	return nil, errors.Errorf("%T has no Go value", el)
}

// ValueT converts an element to a specific Go value.
func ValueT[T any](fnin *FuncInputs, el ir.Element) (valT T, err error) {
	valAny, err := Value(fnin, el)
	if err != nil {
		return
	}
	return cast.To[T](valAny)
}
