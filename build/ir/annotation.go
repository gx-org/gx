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

package ir

import (
	"fmt"
	"maps"
	"slices"
)

type (
	// Annotations maps key to annotation value.
	Annotations struct {
		anns map[string]Annotation
	}

	// Annotation is meta data associated to a key and a IR element (typically functions).
	Annotation interface {
		Key() string
	}

	// AnnotationT is a key pointing to some data.
	AnnotationT[T any] struct {
		// Key used to identify the annotation.
		key string
		// Value of the annotation.
		value T
	}
)

// Set an annotation using its key.
func (anns *Annotations) Set(ann Annotation) {
	if anns.anns == nil {
		anns.anns = make(map[string]Annotation)
	}
	anns.anns[ann.Key()] = ann
}

// Get an annotation given its key. Returns nil if the annotation is not present.
func (anns *Annotations) Get(key string) Annotation {
	return anns.anns[key]
}

// String representation of the annotations.
func (anns *Annotations) String() string {
	return fmt.Sprint(slices.Collect(maps.Keys(anns.anns)))
}

// NewAnnotation creates a new annotation.
// This function is mostly used for tests.
// Prefer Annotations.Append instead.
func NewAnnotation[T any](key string, val T) *AnnotationT[T] {
	return &AnnotationT[T]{
		key:   key,
		value: val,
	}
}

// Key of the annotation.
func (ann *AnnotationT[T]) Key() string {
	return ann.key
}

// Value of the annotation.
func (ann *AnnotationT[T]) Value() T {
	return ann.value
}

// AnnotationFrom returns an annotation given its key.
func AnnotationFrom[T any](f Func, key string) *AnnotationT[T] {
	ann := f.Annotations().Get(key)
	if ann == nil {
		return nil
	}
	return ann.(*AnnotationT[T])
}
