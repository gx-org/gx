// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package annotations provides annotations (i.e. meta-data) to GX functions.
package annotations

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/base/ordered"
)

type (
	// Key is an annotation key.
	// Singleton created by the package defining Annotators.
	Key interface {
		key()
		FullName() string
	}

	key struct {
		fullName string
	}
)

// NewKey returns an annotation key where the name is the type of the given argument.
func NewKey(a any) Key {
	return &key{
		fullName: fmt.Sprintf("%T", a),
	}
}

func (key) key() {}

// FullName of the key.
func (k *key) FullName() string {
	return k.fullName
}

type (
	// Annotations maps key to annotation value.
	Annotations struct {
		anns *ordered.Map[Key, annotation]
	}

	annotation interface {
		Key() Key
		String() string
	}

	annotationT[T any] struct {
		key   Key
		value T
	}

	// Annotated owns a set of annotations.
	Annotated interface {
		ShortString() string
		Annotations() *Annotations
	}
)

// Set an annotation using its key.
func (anns *Annotations) set(ann annotation) bool {
	if anns.anns == nil {
		anns.anns = ordered.NewMap[Key, annotation]()
	}
	_, has := anns.anns.Load(ann.Key())
	if has {
		return false
	}
	anns.anns.Store(ann.Key(), ann)
	return true
}

// Get an annotation given its key. Returns nil if the annotation is not present.
func (anns *Annotations) get(key Key) annotation {
	if anns.anns == nil {
		return nil
	}
	ann, _ := anns.anns.Load(key)
	return ann
}

// String representation of the annotations.
func (anns *Annotations) String() string {
	if anns.anns == nil {
		return ""
	}
	var b strings.Builder
	for k, v := range anns.anns.Iter() {
		b.WriteString(fmt.Sprintf("%s: %s", k.FullName(), v.String()))
	}
	return b.String()
}

func (a annotationT[T]) Key() Key {
	return a.key
}

func (a annotationT[T]) String() string {
	return fmt.Sprint(a.value)
}

// Set an annotation on an annotated receiver.
func Set[T any](rcv Annotated, key Key, val T) error {
	ok := rcv.Annotations().set(&annotationT[T]{
		key:   key,
		value: val,
	})
	if !ok {
		return errors.Errorf("annotation %s has already been defined on %s", key.FullName(), rcv.ShortString())
	}
	return nil
}

// GetDef gets an annotation from an annotated receiver or create a new one if it is not already set.
func GetDef[T any](rcv Annotated, key Key, n func() T) T {
	anns := rcv.Annotations()
	ann := anns.get(key)
	if ann != nil {
		return ann.(*annotationT[T]).value
	}
	val := n()
	Set(rcv, key, val)
	return val
}

// Get gets an annotation from an annotated receiver.
// Returns a zero value if the annotation has not been defined.
func Get[T any](rcv Annotated, key Key) (t T) {
	ann := rcv.Annotations().get(key)
	if ann == nil {
		return
	}
	return ann.(*annotationT[T]).value
}
