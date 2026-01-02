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

// Package fmtarray formats arrays into string.
package fmtarray

import (
	"fmt"
	"strings"

	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/gx/build/ir/irkind"
)

func computeIndex(offsets []int, p []int) int {
	var index int
	for i, v := range p {
		index += int(offsets[i]) * v
	}
	return index
}

type builder[T dtype.GoDataType] struct {
	w       *strings.Builder
	data    []T
	axes    []int
	offsets []int
}

func (b *builder[T]) toValue(x T) string {
	var fmtstr string
	switch any(x).(type) {
	case float32:
		fmtstr = "%.6f"
	case float64:
		fmtstr = "%.10f"
	default:
		return fmt.Sprint(x)
	}

	result := fmt.Sprintf(fmtstr, x)
	if strings.ContainsRune(result, '.') {
		// Remove any number of trailing zeroes after the decimal point, and remove
		// the point itself if there are no digits after it.
		result = strings.TrimRight(result, "0")
		result = strings.TrimSuffix(result, ".")
	}
	return result
}

func (b *builder[T]) printScalar() {
	b.w.WriteString("(")
	b.w.WriteString(b.toValue(b.data[0]))
	b.w.WriteString(")")
}

const skipArrayValues = false

func (b *builder[T]) printVector(p []int) {
	if skipArrayValues {
		b.w.WriteString("{...}")
		return
	}

	fullPos := make([]int, len(b.axes))
	copy(fullPos, p)
	vecSize := b.axes[len(b.axes)-1]

	vec := make([]string, vecSize)
	for i := 0; i < vecSize; i++ {
		fullPos[len(fullPos)-1] = i
		vec[i] = b.toValue(b.data[computeIndex(b.offsets, fullPos)])
	}
	vecS := fmt.Sprintf("{%s}", strings.Join(vec, ", "))
	b.w.WriteString(vecS)
}

func toPosition(parentPosition []int) []int {
	position := append([]int{}, parentPosition...)
	position = append(position, 0)
	return position
}

func (b *builder[T]) printMatrix(indent string, parentPosition []int) {
	numRows := b.axes[len(b.axes)-2]
	position := toPosition(parentPosition)
	if skipArrayValues {
		b.w.WriteString("{...}")
		return
	}
	b.w.WriteString(indent + "{\n")
	for i := 0; i < numRows; i++ {
		b.w.WriteString(indent + tab)
		position[len(position)-1] = i
		b.printVector(position)
		b.w.WriteString(",\n")
	}
	b.w.WriteString(indent + "}")
}

const tab = "\t"

func (b *builder[T]) printRec(indent string, parentPosition []int) {
	if len(b.axes)-len(parentPosition) == 2 {
		b.printMatrix(indent, parentPosition)
		return
	}

	b.w.WriteString(indent + "{\n")
	position := toPosition(parentPosition)
	for i := 0; i < b.axes[len(parentPosition)]; i++ {
		position[len(position)-1] = i
		b.printRec(indent+tab, position)
		b.w.WriteString(",\n")
	}
	b.w.WriteString(indent + "}")
}

func (b *builder[T]) printType() int {
	shapes := make([]string, len(b.axes))
	total := 1
	for i, size := range b.axes {
		shapes[i] = fmt.Sprintf("[%d]", size)
		total *= size
	}
	shapes = append(shapes, irkind.KindGeneric[T]().String())
	b.w.WriteString(strings.Join(shapes, ""))
	return total
}

func axesOffsets(axes []int) []int {
	offsets := make([]int, len(axes))
	for i := range offsets {
		offsets[i] = 1
		for _, d := range axes[i+1:] {
			offsets[i] *= d
		}
	}
	return offsets
}

// Sprint returns a string representation of a tensor.
func Sprint[T dtype.GoDataType](data []T, axes []int) string {
	b := builder[T]{
		w:       &strings.Builder{},
		data:    data,
		axes:    axes,
		offsets: axesOffsets(axes),
	}
	total := b.printType()
	if total != len(data) {
		return fmt.Sprintf("len(data)=%d does not match axes %v=%d", len(data), axes, total)
	}
	switch len(b.axes) {
	case 0:
		b.printScalar()
	case 1:
		b.printVector(nil)
	case 2:
		b.printMatrix("", nil)
	default:
		b.printRec("", nil)
	}
	return b.w.String()
}
