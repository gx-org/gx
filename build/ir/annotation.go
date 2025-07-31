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

type (
	// Annotations maps key to annotation value.
	Annotations struct {
		Anns []*Annotation
	}

	// Annotation is a key pointing to some data.
	Annotation struct {
		// Key used to identify the annotation.
		key string
		// Value of the annotation.
		value any
	}
)

// Append a new annotation to the slice of annotation.
func (anns *Annotations) Append(pkg *Package, key string, value any) {
	anns.AppendAnn(NewAnnotation(pkg.FullName()+":"+key, value))
}

// AppendAnn appends an annotation already constructed.
func (anns *Annotations) AppendAnn(ann *Annotation) {
	anns.Anns = append(anns.Anns, ann)
}

// NewAnnotation creates a new annotation.
// This function is mostly used for tests.
// Prefer Annotations.Append instead.
func NewAnnotation(key string, val any) *Annotation {
	return &Annotation{
		key:   key,
		value: val,
	}
}

// Key of the annotation.
func (ann *Annotation) Key() string {
	return ann.key
}

// Value of the annotation.
func (ann *Annotation) Value() any {
	return ann.value
}
